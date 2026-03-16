package dns

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	CloudflareDotEndpoint    = "1.1.1.1"
	CloudflareDotEndpointSNI = "cloudflare-dns.com"
	GoogleDotEndpoint        = "8.8.8.8"
	GoogleDotEndpointSNI     = "dns.google"

	DefaultDotTimeout = 5 * time.Second
)

type DotUpstream struct {
	address string
	dialer  *tls.Dialer
}

// NewDotUpstream creates new DNS-over-TLS resolver upstream.
//
// NewDotUpstream создает DNS-over-TLS бэкенд для [dns.Resolver].
func NewDotUpstream(address string, insecureSkipVerify bool) *DotUpstream {
	host, port, _ := net.SplitHostPort(address)
	if port == "" {
		host = address
		port = "853"
	}
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{
			Timeout: DefaultDotTimeout,
		},
		Config: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
			ServerName:         host,
		},
	}
	return &DotUpstream{
		address: net.JoinHostPort(host, port),
		dialer:  dialer,
	}
}

func (r *DotUpstream) Resolve(ctx context.Context, req []byte) ([]byte, error) {
	conn, err := r.dialer.Dial("tcp", r.address)
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	defer conn.Close()
	n, err := conn.Write(req)
	if err != nil {
		return nil, err
	}
	if n != len(req) {
		return nil, fmt.Errorf("uncomplete write")
	}
	var lenBuf [2]byte
	n, err = conn.Read(lenBuf[:])
	if err != nil || n != 2 {
		return nil, fmt.Errorf("uncomplete read")
	}
	// response size
	bufSize := int(lenBuf[0])<<8 | int(lenBuf[1])
	// response size + response
	buf := make([]byte, bufSize+2)
	buf[0] = lenBuf[0]
	buf[1] = lenBuf[1]
	read, err := io.ReadFull(conn, buf[2:])
	if err != nil || read != bufSize {
		return nil, fmt.Errorf("uncomplete read")
	}
	return buf, nil
}
