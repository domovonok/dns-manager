package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os/signal"
	"syscall"
	"time"

	dnsv1 "github.com/domovonok/dns-manager/api/dns/v1"
	"github.com/domovonok/dns-manager/internal/config"
	"github.com/domovonok/dns-manager/internal/handler/dns"
	"github.com/domovonok/dns-manager/internal/repository/resolvconf"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg, err := config.NewServer()
	if err != nil {
		log.Fatalln("can not initialize config:", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalln("can not initialize logger:", err)
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			log.Println("failed to sync logger:", err)
		}
	}(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	repo := resolvconf.New(cfg.ResolveConfPath)

	handler := dns.New(logger, repo)

	if err := runGrpc(ctx, cfg, logger, handler); err != nil {
		logger.Fatal("grpc server error", zap.Error(err))
	}
}

func runGrpc(ctx context.Context, cfg *config.Server, logger *zap.Logger, handler *dns.Handler) error {
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	reflection.Register(s)

	dnsv1.RegisterDnsServiceServer(s, handler)
	logger.Info("grpc server listening at addr", zap.String("addr", lis.Addr().String()))

	serveErr := make(chan error, 1)
	go func() {
		err := s.Serve(lis)
		if errors.Is(err, grpc.ErrServerStopped) {
			err = nil
		}
		serveErr <- err
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		logger.Info("grpc server shutdown started", zap.Duration("timeout", cfg.GracefulShutdownTimeout))

		stopped := make(chan struct{})
		go func() {
			s.GracefulStop()
			close(stopped)
		}()

		timer := time.NewTimer(cfg.GracefulShutdownTimeout)
		defer timer.Stop()

		select {
		case <-stopped:
			logger.Info("grpc server stopped gracefully")
		case <-timer.C:
			logger.Warn("grpc server graceful shutdown timeout exceeded, forcing stop")
			s.Stop()
			<-stopped
		}

		return <-serveErr
	}
}
