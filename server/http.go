package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type HttpListener struct {
	server *Server
}

func NewHttpListener() *HttpListener {
	return &HttpListener{}
}

func (h *HttpListener) Name() string {
	return "http"
}

func (h *HttpListener) handleTunnel(w http.ResponseWriter, r *http.Request) {
	source, _, _ := net.SplitHostPort(r.RemoteAddr)
	destination, _, _ := net.SplitHostPort(r.Host)
	if destination == "mc.yandex.ru" || destination == "counter.yadro.ru" {
		slog.InfoContext(r.Context(), "[https] blocked", slog.String("destination", destination))
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	slog.InfoContext(r.Context(), "[https] handle tunnel",
		slog.String("source", source),
		slog.String("destination", destination))
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	go h.transfer(destConn, clientConn)
	go h.transfer(clientConn, destConn)
}

func (h *HttpListener) transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}

func (h *HttpListener) handleHttp(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		h.handleTunnel(w, r)
		return
	}
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}

func (h *HttpListener) ListenAndServe(ctx context.Context, ln net.Listener, s *Server) error {
	h.server = s
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.handleHttp(w, r)
		}),
	}
	go func() {
		<-ctx.Done()
		server.Close()
		slog.InfoContext(ctx, "[http] stopped", slog.String("server", s.address))
	}()

	slog.InfoContext(ctx, "[http] started", slog.String("server", s.address))
	return server.Serve(ln)
}
