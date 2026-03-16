package dns

import (
	"context"
	"net"
	"sync"
	"testing"
)

func testDohResolver(t *testing.T) {
	var dohEndpoints = []string{CloudflareDohEndpoint, GoogleDohEndpoint}
	for _, endpoint := range dohEndpoints {
		resolver := NewResolver(NewDohUpstream(endpoint))
		testResolver(t, resolver.ToDnsResolver(), endpoint)
	}
}

func testDotResolver(t *testing.T) {
	var dotEndpoints = []string{CloudflareDotEndpoint, GoogleDotEndpoint, "dns.google"}
	for _, endpoint := range dotEndpoints {
		resolver := NewResolver(NewDotUpstream(endpoint, false))
		testResolver(t, resolver.ToDnsResolver(), endpoint)
	}
}

func TestChainedResolver(t *testing.T) {
	resolver := NewResolver(NewChainedUpstream(
		//NewDohUpstream(CloudflareDohEndpoint),
		NewDotUpstream(GoogleDotEndpoint, false),
	))
	r := resolver.ToDnsResolver()
	//testResolver(t, r, "chained")
	_, err := r.LookupHost(context.Background(), "рус.сайт.фк.рнк.ржд.ошибка.рф")
	if err == nil {
		t.Fatal("expected error, got none")
	}
}

func testResolver(t *testing.T, r *net.Resolver, endpoint string) {
	var testHosts = []string{"google.com", "youtube.com"}
	for _, host := range testHosts {
		var wg sync.WaitGroup
		for i := range 3 {
			wg.Go(func() {
				addrs, err := r.LookupHost(context.Background(), host)
				if err != nil {
					t.Fatalf("[iter %d] could not lookup host %s with %s: %v", i, host, endpoint, err)
				}
				if len(addrs) == 0 {
					t.Fatalf("[iter %d] no addresses for host %s with %s: %v", i, host, endpoint, err)
				}
				for _, addr := range addrs {
					hosts, err := r.LookupAddr(context.Background(), addr)
					if err != nil || len(hosts) == 0 {
						t.Fatalf("[iter %d] could not lookup addr %s with %s: %v", i, host, endpoint, err)
					}
				}
			})
		}
		wg.Wait()
	}
}

func testUpdResolver(t *testing.T) {
	endpoint := "" // "127.0.0.53:53"
	host := "localhost"
	r := NewResolver(NewUdpUpstream(endpoint)).ToDnsResolver()
	var wg sync.WaitGroup
	for i := range 3 {
		wg.Go(func() {
			addrs, err := r.LookupHost(context.Background(), host)
			if err != nil {
				t.Fatalf("[iter %d] could not lookup host %s with %s: %v", i, host, endpoint, err)
			}
			if len(addrs) == 0 {
				t.Fatalf("[iter %d] no addresses for host %s with %s: %v", i, host, endpoint, err)
			}
			for _, addr := range addrs {
				hosts, err := r.LookupAddr(context.Background(), addr)
				if err != nil || len(hosts) == 0 {
					t.Fatalf("[iter %d] could not lookup addr %s with %s: %v", i, host, endpoint, err)
				}
			}
		})
	}
	wg.Wait()
}
