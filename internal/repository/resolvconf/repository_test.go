package resolvconf_test

import (
	"context"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	xerrors "github.com/domovonok/dns-manager/internal/errors"
	"github.com/domovonok/dns-manager/internal/repository/resolvconf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []netip.Addr
		wantErr error
	}{
		{
			name: "success",
			content: "search example.org\n" +
				"nameserver 192.0.2.1\n" +
				"nameserver 2001:db8::1\n" +
				"# nameserver 203.0.113.1\n",
			want: []netip.Addr{
				netip.MustParseAddr("192.0.2.1"),
				netip.MustParseAddr("2001:db8::1"),
			},
		},
		{
			name:    "wrong ip format",
			content: "nameserver not-an-ip\n",
			wantErr: xerrors.ErrWrongIPFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := writeResolvConf(t, tt.content)
			repo := resolvconf.New(path)

			got, err := repo.List(context.Background())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRepositoryListReadError(t *testing.T) {
	t.Parallel()

	repo := resolvconf.New(filepath.Join(t.TempDir(), "missing.conf"))

	got, err := repo.List(context.Background())

	require.ErrorIs(t, err, xerrors.ErrReadError)
	assert.Nil(t, got)
}

func TestRepositoryAdd(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("2001:db8::1")

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "appends after trailing newline",
			content: "search example.org\n",
			want:    "search example.org\nnameserver 2001:db8::1\n",
		},
		{
			name:    "adds missing separator newline",
			content: "search example.org",
			want:    "search example.org\nnameserver 2001:db8::1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := writeResolvConf(t, tt.content)
			repo := resolvconf.New(path)

			err := repo.Add(context.Background(), ip)

			require.NoError(t, err)
			assert.Equal(t, tt.want, readFile(t, path))
		})
	}
}

func TestRepositoryAddAlreadyExists(t *testing.T) {
	t.Parallel()

	content := "nameserver 192.0.2.1\n"
	path := writeResolvConf(t, content)
	repo := resolvconf.New(path)

	err := repo.Add(context.Background(), netip.MustParseAddr("192.0.2.1"))

	require.ErrorIs(t, err, xerrors.ErrAlreadyExist)
	assert.Equal(t, content, readFile(t, path))
}

func TestRepositoryRemove(t *testing.T) {
	t.Parallel()

	path := writeResolvConf(t, "search example.org\n"+
		"nameserver 192.0.2.1\n"+
		"nameserver 2001:db8::1\n"+
		"# comment\n")
	repo := resolvconf.New(path)

	err := repo.Remove(context.Background(), netip.MustParseAddr("192.0.2.1"))

	require.NoError(t, err)
	assert.Equal(t, "search example.org\nnameserver 2001:db8::1\n# comment\n", readFile(t, path))
}

func TestRepositoryRemoveWrongIPFormat(t *testing.T) {
	t.Parallel()

	content := "nameserver not-an-ip\n"
	path := writeResolvConf(t, content)
	repo := resolvconf.New(path)

	err := repo.Remove(context.Background(), netip.MustParseAddr("192.0.2.1"))

	require.ErrorIs(t, err, xerrors.ErrWrongIPFormat)
	assert.Equal(t, content, readFile(t, path))
}

func TestRepositoryAddReadError(t *testing.T) {
	t.Parallel()

	repo := resolvconf.New(filepath.Join(t.TempDir(), "missing.conf"))

	err := repo.Add(context.Background(), netip.MustParseAddr("192.0.2.1"))

	require.ErrorIs(t, err, xerrors.ErrReadError)
}

func writeResolvConf(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "resolv.conf")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(data)
}

func TestRepositoryRemoveReadError(t *testing.T) {
	t.Parallel()

	repo := resolvconf.New(filepath.Join(t.TempDir(), "missing.conf"))

	err := repo.Remove(context.Background(), netip.MustParseAddr("192.0.2.1"))

	require.ErrorIs(t, err, xerrors.ErrReadError)
}
