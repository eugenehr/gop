package server

import (
	"context"
	"testing"
)

func TestHttp(t *testing.T) {
	server, err := NewServerFromURL("http://:13128")
	if err != nil {
		t.Fatal(err)
	}
	server.ListenAndServe(context.Background(), nil)
}
