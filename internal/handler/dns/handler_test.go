package dns_test

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	dnsv1 "github.com/domovonok/dns-manager/api/dns/v1"
	xerrors "github.com/domovonok/dns-manager/internal/errors"
	"github.com/domovonok/dns-manager/internal/handler/dns"
	"github.com/domovonok/dns-manager/internal/handler/dns/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestHandlerAdd(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.0.2.1")

	tests := []struct {
		name     string
		req      *dnsv1.AddRequest
		repoErr  error
		wantCode codes.Code
		wantCall bool
	}{
		{
			name:     "success",
			req:      &dnsv1.AddRequest{Ip: ip.String()},
			wantCode: codes.OK,
			wantCall: true,
		},
		{
			name:     "invalid ip",
			req:      &dnsv1.AddRequest{Ip: "not-an-ip"},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "already exists",
			req:      &dnsv1.AddRequest{Ip: ip.String()},
			repoErr:  xerrors.ErrAlreadyExist,
			wantCode: codes.AlreadyExists,
			wantCall: true,
		},
		{
			name:     "unknown repo error",
			req:      &dnsv1.AddRequest{Ip: ip.String()},
			repoErr:  errors.New("disk is unavailable"),
			wantCode: codes.Internal,
			wantCall: true,
		},
	}

	for _, tt := range tests {
		test(t, tt.name, func(t *testing.T, ctx context.Context, repo *mocks.Mockrepo, handler *dns.Handler) {
			if tt.wantCall {
				repo.EXPECT().Add(ctx, ip).Return(tt.repoErr)
			}

			res, err := handler.Add(ctx, tt.req)
			require.Equal(t, tt.wantCode, status.Code(err))
			if err == nil {
				require.NotNil(t, res)
			}
		})
	}
}

func TestHandlerRemove(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("2001:db8::1")

	tests := []struct {
		name     string
		req      *dnsv1.RemoveRequest
		repoErr  error
		wantCode codes.Code
		wantCall bool
	}{
		{
			name:     "success",
			req:      &dnsv1.RemoveRequest{Ip: ip.String()},
			wantCode: codes.OK,
			wantCall: true,
		},
		{
			name:     "invalid ip",
			req:      &dnsv1.RemoveRequest{Ip: "not-an-ip"},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "wrong ip format from repo",
			req:      &dnsv1.RemoveRequest{Ip: ip.String()},
			repoErr:  xerrors.ErrWrongIPFormat,
			wantCode: codes.FailedPrecondition,
			wantCall: true,
		},
		{
			name:     "unknown repo error",
			req:      &dnsv1.RemoveRequest{Ip: ip.String()},
			repoErr:  errors.New("write failed"),
			wantCode: codes.Internal,
			wantCall: true,
		},
	}

	for _, tt := range tests {
		test(t, tt.name, func(t *testing.T, ctx context.Context, repo *mocks.Mockrepo, handler *dns.Handler) {
			if tt.wantCall {
				repo.EXPECT().Remove(ctx, ip).Return(tt.repoErr)
			}

			res, err := handler.Remove(ctx, tt.req)
			require.Equal(t, tt.wantCode, status.Code(err))
			if err == nil {
				require.NotNil(t, res)
			}
		})
	}
}

func TestHandlerList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ips      []netip.Addr
		repoErr  error
		want     []string
		wantCode codes.Code
	}{
		{
			name: "success",
			ips: []netip.Addr{
				netip.MustParseAddr("192.0.2.1"),
				netip.MustParseAddr("2001:db8::1"),
			},
			want:     []string{"192.0.2.1", "2001:db8::1"},
			wantCode: codes.OK,
		},
		{
			name:     "read error",
			repoErr:  xerrors.ErrReadError,
			wantCode: codes.FailedPrecondition,
		},
		{
			name:     "unknown repo error",
			repoErr:  errors.New("read failed"),
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		test(t, tt.name, func(t *testing.T, ctx context.Context, repo *mocks.Mockrepo, handler *dns.Handler) {
			repo.EXPECT().List(ctx).Return(tt.ips, tt.repoErr)

			res, err := handler.List(ctx, &emptypb.Empty{})
			require.Equal(t, tt.wantCode, status.Code(err))
			if err != nil {
				return
			}

			assert.Equal(t, tt.want, res.GetIps())
		})
	}
}

func test(
	t *testing.T,
	name string,
	run func(t *testing.T, ctx context.Context, repo *mocks.Mockrepo, handler *dns.Handler),
) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repo := mocks.NewMockrepo(ctrl)
		handler := dns.New(zap.NewNop(), repo)

		run(t, context.Background(), repo, handler)
	})
}
