package dns

import (
	"context"
)

type ChainedUpstream struct {
	chain []ResolverUpstream
}

// NewChainedUpstream creates a backend from multiple [dns.ResolverUpstreams],
// // which are used sequentially until one resolves the requested address.
//
// NewChainedUpstream создает бэкенд из нескольких [dns.ResolverUpstream],
// которые используются последовательно, пока один из них не разрешит запрашиваемый адрес.
func NewChainedUpstream(upstreams ...ResolverUpstream) *ChainedUpstream {
	return &ChainedUpstream{
		chain: upstreams,
	}
}

func (r *ChainedUpstream) Resolve(ctx context.Context, req []byte) (resp []byte, err error) {
	for _, upstream := range r.chain {
		resp, err = upstream.Resolve(ctx, req)
		if err == nil {
			return resp, nil
		}
	}
	return nil, err
}
