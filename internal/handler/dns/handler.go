package dns

import (
	"context"
	"errors"
	"net/netip"

	dnsv1 "github.com/domovonok/dns-manager/api/dns/v1"
	xerrors "github.com/domovonok/dns-manager/internal/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockgen -source=handler.go -destination=mocks/handler_mocks.go -package=mocks
type repo interface {
	Add(ctx context.Context, ip netip.Addr) error
	Remove(ctx context.Context, ip netip.Addr) error
	List(ctx context.Context) ([]netip.Addr, error)
}

type Handler struct {
	dnsv1.UnimplementedDnsServiceServer

	logger *zap.Logger
	repo   repo
}

func New(logger *zap.Logger, repo repo) *Handler {
	return &Handler{logger: logger, repo: repo}
}

func (h *Handler) Add(ctx context.Context, req *dnsv1.AddRequest) (*emptypb.Empty, error) {
	h.logger.Info("add request received", zap.String("ip", req.GetIp()))

	if err := req.Validate(); err != nil {
		h.logger.Warn("add request validation failed", zap.String("ip", req.GetIp()), zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ip, err := netip.ParseAddr(req.GetIp())
	if err != nil {
		h.logger.Error("failed to parse ip, validation broke", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := h.repo.Add(ctx, ip); err != nil {
		grpcErr := h.mapErrorToGRPC(err)
		h.logger.Warn(
			"add request failed",
			zap.String("ip", ip.String()),
			zap.String("code", status.Code(grpcErr).String()),
			zap.Error(err),
		)
		return nil, grpcErr
	}

	h.logger.Info("add request completed", zap.String("ip", ip.String()))

	return &emptypb.Empty{}, nil
}

func (h *Handler) Remove(ctx context.Context, req *dnsv1.RemoveRequest) (*emptypb.Empty, error) {
	h.logger.Info("remove request received", zap.String("ip", req.GetIp()))

	if err := req.Validate(); err != nil {
		h.logger.Warn("remove request validation failed", zap.String("ip", req.GetIp()), zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ip, err := netip.ParseAddr(req.GetIp())
	if err != nil {
		h.logger.Error("failed to parse ip, validation broke", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := h.repo.Remove(ctx, ip); err != nil {
		grpcErr := h.mapErrorToGRPC(err)
		h.logger.Warn(
			"remove request failed",
			zap.String("ip", ip.String()),
			zap.String("code", status.Code(grpcErr).String()),
			zap.Error(err),
		)
		return nil, grpcErr
	}

	h.logger.Info("remove request completed", zap.String("ip", ip.String()))

	return &emptypb.Empty{}, nil
}

func (h *Handler) List(ctx context.Context, _ *emptypb.Empty) (*dnsv1.ListResponse, error) {
	h.logger.Info("list request received")

	ips, err := h.repo.List(ctx)
	if err != nil {
		grpcErr := h.mapErrorToGRPC(err)
		h.logger.Warn(
			"list request failed",
			zap.String("code", status.Code(grpcErr).String()),
			zap.Error(err),
		)
		return nil, grpcErr
	}

	res := make([]string, len(ips))
	for i, ip := range ips {
		res[i] = ip.String()
	}

	h.logger.Info("list request completed", zap.Int("count", len(res)))

	return &dnsv1.ListResponse{Ips: res}, nil
}

func (h *Handler) mapErrorToGRPC(err error) error {
	code := getGRPCCode(err)
	message := err.Error()
	if code == codes.Internal {
		h.logger.Error("internal error", zap.Error(err))
		message = "internal error"
	}
	return status.Error(code, message)
}

func getGRPCCode(err error) codes.Code {
	switch {
	case errors.Is(err, xerrors.ErrAlreadyExist):
		return codes.AlreadyExists
	case errors.Is(err, xerrors.ErrWrongIPFormat):
		return codes.FailedPrecondition
	case errors.Is(err, xerrors.ErrReadError):
		return codes.FailedPrecondition
	default:
		return codes.Internal
	}
}
