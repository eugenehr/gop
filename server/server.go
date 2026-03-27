package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"
)

// Handler incoming connection handler interface.
//
// Handler интерфейс обработчика входящих соединений.
type Handler interface {
	Name() string
	HandleConn(context.Context, net.Conn, *Server)
}

// Listener interface implementors handles accepting incoming connections.
//
// Listener интерфейс слушателя входящих соединений.
type Listener interface {
	ListenAndServe(context.Context, net.Listener, *Server) error
}

// Server struct holds proxy server configuration and its dependencies.
//
// Server структура для хранения настроек сервера и зависимостей, необходимых для его работы.
type Server struct {
	address   string
	listener  Listener
	tlsConfig *tls.Config
}

// NewServerFromURL creates a new Server instance from given URL.
//
// NewServerFromURL создает новый экземпляр сервера на основе переданного URL.
// URL может быть одного из следующих вариантов:
// 1. http[s]://[host]:port[?certfile=path_to_cert.pem&keyfile=path_to_key.pem&cacertfile=path_to_ca_cert.pem]
// 2. socks[5]://[host]:port[?certfile=path_to_cert.pem&keyfile=path_to_key.pem&cacertfile=path_to_ca_cert.pem]
// 3. ss://method:password@[host]:port
func NewServerFromURL(rawUrl string) (*Server, error) {
	parsed, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	scheme, address := strings.ToLower(parsed.Scheme), strings.ToLower(parsed.Host)

	var listener Listener
	switch scheme {
	case "socks", "socks5":
		listener = &handlerListener{handler: NewSocks5Handler()}
	case "http", "https":
		listener = NewHttpListener()
	case "ss", "shadowsocks":
		return nil, errors.New("not implemented")
	default:
		return nil, errors.New("unknown scheme")
	}
	// tls config
	var tlsConfig *tls.Config
	certfile := parsed.Query().Get("certfile")
	keyfile := parsed.Query().Get("keyfile")
	cacertfile := parsed.Query().Get("cacertfile")

	if certfile != "" {
		if keyfile == "" {
			return nil, errors.New("no tls key file")
		}
		cert, err := tls.LoadX509KeyPair(certfile, keyfile)
		if err != nil {
			return nil, err
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		if cacertfile != "" {
			caCert, err := os.ReadFile(cacertfile)
			if err != nil {
				return nil, err
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.ClientCAs = caCertPool
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
	} else {
		if scheme == "https" {
			return nil, errors.New("no tls cert file")
		}
		if keyfile != "" {
			slog.Warn("keyfile ignored: no tls cert file")
		}
		if cacertfile != "" {
			slog.Warn("cacertfile ignored: no tls cert file")
		}
	}
	return NewServer(address, listener, tlsConfig), nil
}

// NewServer creates a new Server instance with given connection handler.
//
// NewServer создает новый экземпляр сервера с обработчиком входящих соединений.
func NewServer(address string, listener Listener, tlsConfig *tls.Config) *Server {
	return &Server{
		address:   address,
		listener:  listener,
		tlsConfig: tlsConfig,
	}
}

// ListenAndServe starts client connections handling.
// Parameter addrCh is used for debug purposes only and allows the callers
// to obtain actual address and port that the server is listening on.
//
// ListenAndServe запускает обработку сетевых запросов.
// Параметр addrCh используется только для отладочных целей и позволяет получить вызывающей стороне
// реальные адрес и порт, которые прослушивает сервер.
func (s *Server) ListenAndServe(ctx context.Context, addrCh chan<- string) (err error) {
	var ln net.Listener
	if s.tlsConfig == nil {
		ln, err = net.Listen("tcp", s.address)
	} else {
		ln, err = tls.Listen("tcp", s.address, s.tlsConfig)
	}
	if err != nil {
		return err
	}

	s.address = ln.Addr().String()
	if addrCh != nil {
		addrCh <- s.address
	}

	return s.listener.ListenAndServe(ctx, ln, s)
}

type handlerListener struct {
	handler Handler
}

func (n *handlerListener) ListenAndServe(ctx context.Context, ln net.Listener, s *Server) error {
	defer ln.Close()
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				slog.InfoContext(ctx, fmt.Sprintf("[%s] %s", n.handler.Name(), "stopped"),
					slog.String("server", s.address))
				return ctx.Err()
			default:
				slog.ErrorContext(ctx, fmt.Sprintf("[%s] %s", n.handler.Name(), "could not accept"),
					slog.String("server", s.address),
					slog.Any("error", err))
				return err
			}
		}
		go n.handler.HandleConn(ctx, conn, s)
	}
}
