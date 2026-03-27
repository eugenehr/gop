package server

import (
	"context"
	"log/slog"
	"net"
	"time"
)

type Socks5Handler struct {
}

func NewSocks5Handler() *Socks5Handler {
	return &Socks5Handler{}
}

func (h *Socks5Handler) Name() string {
	return "socks5"
}

func (h *Socks5Handler) HandleConn(ctx context.Context, conn net.Conn, s *Server) {
	slog.InfoContext(ctx, "[socks5] connection accepted",
		slog.String("server", s.address),
		slog.String("client", conn.RemoteAddr().String()),
	)
	startTime := time.Now()
	defer func() {
		slog.InfoContext(ctx, "[socks5] connection closed",
			slog.String("server", s.address),
			slog.String("client", conn.RemoteAddr().String()),
			slog.Any("duration", time.Since(startTime)))
		_ = conn.Close()
	}()
	time.Sleep(3 * time.Second)
}
