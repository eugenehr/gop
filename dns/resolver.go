package dns

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

type cacheEntry struct {
	resp []byte
	exp  time.Time
}

type cacheMap map[string]*cacheEntry

type inflightEntry struct {
	wait chan struct{}
	resp []byte
	err  error
}

// ResolverUpstream interface for upstream implementations.
//
// ResolverUpstream для подключения сторонних бэкендов.
type ResolverUpstream interface {
	Resolve(ctx context.Context, req []byte) ([]byte, error)
}

// Resolver [dns.Resolver] interface.
type Resolver struct {
	mx       sync.Mutex
	cache    atomic.Pointer[cacheMap]
	inflight map[string]*inflightEntry
	upstream ResolverUpstream
}

// NewResolver creates new [dns.Resolver] with given upstream.
//
// NewResolver создает новый [dns.Resolver] с указанным бэкендом.
func NewResolver(upstream ResolverUpstream) *Resolver {
	resolver := Resolver{
		inflight: map[string]*inflightEntry{},
		upstream: upstream,
	}
	cache := make(cacheMap)
	resolver.cache.Store(&cache)
	return &resolver
}

// Resolve resolves raw DNS response.
func (r *Resolver) Resolve(ctx context.Context, req []byte) (resp []byte, err error) {
	// try to find cached response
	key, err := r.key(req[2:])
	if err != nil {
		return nil, err
	}
	cachePtr := r.cache.Load()
	cached, ok := (*cachePtr)[key]
	if ok {
		if time.Now().Before(cached.exp) {
			resp = make([]byte, len(cached.resp))
			copy(resp, cached.resp)
			resp[2] = req[2]
			resp[3] = req[3]
			return resp, nil
		}
	}
	// if no response in the cache then resolve request with upstream
	inf, leader := r.startInflight(key)
	if !leader {
		// other goroutine is resolving this request. wait for it
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-inf.wait:
			if inf.err != nil {
				return nil, inf.err
			}
			resp = make([]byte, len(inf.resp))
			copy(resp, inf.resp)
			resp[2] = req[2]
			resp[3] = req[3]
			return resp, nil
		}
	}

	resp, err = r.upstream.Resolve(ctx, req)
	if err == nil {
		// extract TTL and put response to the cache
		ttl := r.ttl(resp[2:])
		cachedResp := make([]byte, len(resp))
		copy(cachedResp, resp)

		cached = &cacheEntry{
			resp: cachedResp,
			exp:  time.Now().Add(time.Duration(ttl) * time.Second),
		}
		r.cleanupAndSet(key, cached)
	}
	r.endInflight(key, resp, err)
	return resp, err
}

// startInflight creates a new or returns an existing inflight structure and a leader flag.
// If the leader flag is true, the calling goroutine is responsible for calling the backend to resolve
// the address. A false leader flag means that another goroutine is already requesting the address,
// and the calling goroutine must wait.
//
// startInflight создает новую или возвращает существующую inflight структуру и признак лидера.
// Если признак лидера true, то вызывающая горутина ответственна за вызов бэкенда для определения
// адреса. Признак лидера false означает, что запрос на адрес уже выполняется другой горутиной, и
// вызывающая горутина должна ждать.
func (r *Resolver) startInflight(key string) (inflight *inflightEntry, leader bool) {
	r.mx.Lock()
	defer r.mx.Unlock()

	if entry, ok := r.inflight[key]; ok {
		return entry, false
	}
	entry := &inflightEntry{
		wait: make(chan struct{}),
	}
	r.inflight[key] = entry
	return entry, true
}

func (r *Resolver) endInflight(key string, resp []byte, err error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	entry := r.inflight[key]
	delete(r.inflight, key)

	entry.resp = resp
	entry.err = err
	close(entry.wait)
}

func (r *Resolver) key(req []byte) (string, error) {
	var p dnsmessage.Parser
	if _, err := p.Start(req); err != nil {
		return "", err
	}
	q, err := p.Question()
	if err != nil {
		return "", err
	}
	return q.Name.String() + "|" + q.Type.String(), nil
}

// ttl extracts minimum TTL from upstream response.
func (r *Resolver) ttl(resp []byte) uint32 {
	var p dnsmessage.Parser
	if _, err := p.Start(resp); err != nil {
		return 60
	}
	if err := p.SkipAllQuestions(); err != nil {
		return 60
	}
	ttl := uint32(3600)
	for {
		h, err := p.AnswerHeader()
		if err != nil {
			break
		}
		if h.TTL > 0 && h.TTL < ttl {
			ttl = h.TTL
		}
		if err = p.SkipAnswer(); err != nil {
			break
		}
	}
	return ttl
}

// cleanup removes expired cached entries.
func (r *Resolver) cleanup() {
	r.cleanupAndSet("", nil)
}

// cleanup removes expired cached entries and put a new entry into the cache.
func (r *Resolver) cleanupAndSet(key string, entry *cacheEntry) {
	for {
		cachePtr := r.cache.Load()
		oldCache := *cachePtr
		newCache := make(cacheMap)
		for k, v := range oldCache {
			if time.Now().Before(v.exp) {
				newCache[k] = v
			}
		}
		if entry != nil {
			newCache[key] = entry
		}
		if r.cache.CompareAndSwap(cachePtr, &newCache) {
			return
		}
	}
}
