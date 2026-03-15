package dns

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"time"
)

const (
	CloudflareDohEndpoint = "https://cloudflare-dns.com/dns-query"
	GoogleDohEndpoint     = "https://dns.google/dns-query"

	DefaultDohTimeout = 5 * time.Second
)

type DohUpstream struct {
	url    string
	client *http.Client
}

// NewDohUpstream creates new DNS-over-HTTPS resolver upstream.
//
// NewDohUpstream создает DNS-over-HTTPS бэкенд для [dns.Resolver].
func NewDohUpstream(url string) *DohUpstream {
	client := http.Client{
		Timeout: DefaultDohTimeout,
	}
	return &DohUpstream{
		url:    url,
		client: &client,
	}
}

func (r *DohUpstream) Resolve(ctx context.Context, req []byte) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(req[2:]))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/dns-message")
	httpReq.Header.Set("Accept", "application/dns-message")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	resp := make([]byte, len(body)+2)
	// response size
	binary.BigEndian.PutUint16(resp, uint16(len(body)))
	// response
	copy(resp[2:], body)
	return resp, err
}
