package dns

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	DefaultUdpTimeout = 5 * time.Second
)

type UdpUpstream struct {
	address string
	dialer  *net.Dialer
}

// NewUdpUpstream creates new standard UDP DNS resolver upstream.
//
// NewUdpUpstream создает бэкенд для [dns.Resolver] с использованием стандартного DNS сервера.
func NewUdpUpstream(address string) *UdpUpstream {
	if address == "" {
		// if address not set then request for system default dns resolver
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
				address = addr
				return nil, errors.New("stop")
			},
		}
		_, _ = r.LookupHost(context.Background(), "google.com")
	}
	host, port, _ := net.SplitHostPort(address)
	if port == "" {
		host = address
		port = "53"
	}
	return &UdpUpstream{
		address: net.JoinHostPort(host, port),
		dialer: &net.Dialer{
			Timeout: DefaultUdpTimeout,
		},
	}
}

func (r *UdpUpstream) Resolve(ctx context.Context, req []byte) ([]byte, error) {
	conn, err := r.dialer.DialContext(ctx, "udp", r.address)
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	defer conn.Close()
	n, err := conn.Write(req[2:])
	if err != nil {
		return nil, err
	}
	if n != len(req)-2 {
		return nil, fmt.Errorf("uncomplete write")
	}
	var buf [1502]byte
	n, err = conn.Read(buf[2:])
	if err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint16(buf[:], uint16(n))
	return buf[:n+2], nil
}
