package resolvconf

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"strings"
	"sync"

	xerrors "github.com/domovonok/dns-manager/internal/errors"
)

type Repository struct {
	mu             sync.Mutex
	resolvConfPath string
}

func New(resolvConfPath string) *Repository {
	return &Repository{resolvConfPath: resolvConfPath}
}

func (r *Repository) Add(_ context.Context, ip netip.Addr) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.resolvConfPath)
	if err != nil {
		return errors.Join(xerrors.ErrReadError, err)
	}

	ips, err := parse(data)
	if err != nil {
		return err
	}

	for _, current := range ips {
		if current == ip {
			return xerrors.ErrAlreadyExist
		}
	}

	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	data = append(data, []byte("nameserver "+ip.String()+"\n")...)

	return os.WriteFile(r.resolvConfPath, data, 0o644)
}

func (r *Repository) Remove(_ context.Context, ip netip.Addr) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.resolvConfPath)
	if err != nil {
		return errors.Join(xerrors.ErrReadError, err)
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nameserver" {
			current, err := netip.ParseAddr(fields[1])
			if err != nil {
				return errors.Join(xerrors.ErrWrongIPFormat, err)
			}
			if current == ip {
				continue
			}
		}

		lines = append(lines, line)
	}

	return os.WriteFile(r.resolvConfPath, []byte(strings.Join(lines, "\n")), 0o644)
}

func (r *Repository) List(_ context.Context) ([]netip.Addr, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.resolvConfPath)
	if err != nil {
		return nil, errors.Join(xerrors.ErrReadError, err)
	}

	return parse(data)
}

func parse(data []byte) ([]netip.Addr, error) {
	var ips []netip.Addr
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "nameserver" {
			continue
		}

		ip, err := netip.ParseAddr(fields[1])
		if err != nil {
			return nil, errors.Join(xerrors.ErrWrongIPFormat, err)
		}
		ips = append(ips, ip)
	}

	return ips, nil
}
