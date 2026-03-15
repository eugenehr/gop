// Package dns contains local caching DNS server implementation with DOH, DOT and UDP upstreams.
//
// Пакет dns содержит реализации кэширующего DNS сервера, работающего поверх
// реализаций DNS-over-HTTPS, DNS-over-TLS и стандартного DNS-сервера.
package dns

import (
	"context"
	"io"
	"net"
	"time"
)

type dnsResolver struct {
	ctx      context.Context
	resolver *Resolver
	resp     []byte
	ptr      int
}

// ToDnsResolver creates [net.Resolver] from [dns.Resolver] for using with standard GO services.
//
// ToDnsResolver создает стандартный [net.Resolver] из [dns.Resolver] для использования его стандартными пакетами GO.
func (r *Resolver) ToDnsResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return &dnsResolver{
				ctx:      ctx,
				resolver: r,
			}, nil
		},
	}
}

func (r *dnsResolver) Write(b []byte) (int, error) {
	resp, err := r.resolver.Resolve(r.ctx, b)
	if err != nil {
		return 0, err
	}
	r.resp = resp
	r.ptr = 0
	return len(b), nil
}

func (r *dnsResolver) Read(b []byte) (int, error) {
	if r.resp == nil || r.ptr >= len(r.resp) {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(b, r.resp[r.ptr:])
	r.ptr += n
	return n, nil
}

func (r *dnsResolver) Close() error {
	return nil
}

func (r *dnsResolver) LocalAddr() net.Addr {
	return nil
}

func (r *dnsResolver) RemoteAddr() net.Addr {
	return nil
}

func (r *dnsResolver) SetDeadline(t time.Time) (err error) {
	if err = r.SetReadDeadline(t); err == nil {
		err = r.SetWriteDeadline(t)
	}
	return err
}

func (r *dnsResolver) SetReadDeadline(time.Time) error {
	return nil
}

func (r *dnsResolver) SetWriteDeadline(time.Time) error {
	return nil
}
