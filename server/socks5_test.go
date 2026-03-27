package server

import (
	"context"
	"net"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

func TestSocks5(t *testing.T) {
	server, err := NewServerFromURL("socks5://127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addrChan := make(chan string)

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		return server.ListenAndServe(ctx, addrChan)
	})
	addr := <-addrChan

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	grp.Wait()
}
