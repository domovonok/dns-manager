package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"syscall"

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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	repo := resolvconf.New(cfg.ResolveConfPath)

	handler := dns.New(logger, repo)

	go runGrpc(cfg, logger, handler)

	<-ctx.Done()
}

func runGrpc(cfg *config.Server, logger *zap.Logger, handler *dns.Handler) {
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Fatal("can not open tcp socket", zap.Error(err))
	}

	s := grpc.NewServer()
	reflection.Register(s)

	dnsv1.RegisterDnsServiceServer(s, handler)
	logger.Info("grpc server listening at addr", zap.String("addr", lis.Addr().String()))

	if err := s.Serve(lis); err != nil {
		logger.Error("grpc server listen error", zap.Error(err))
	}
}
