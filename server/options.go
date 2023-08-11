package server

import (
	"crypto/tls"
	"time"

	"github.com/jursonmo/subpub/session"
)

type ServerOption func(o *Server)

func WithNetwork(network string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

func WithTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

func WithName(name string) ServerOption {
	return func(s *Server) {
		s.name = name
	}
}

func WithTLSConfig(tc *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConf = tc
	}
}

func WithPathHandler(path string, h WsHandler) ServerOption {
	return func(s *Server) {
		s.pathHandlers[path] = h
	}
}

func WithOnConnect(h func(session.Sessioner)) ServerOption {
	return func(s *Server) {
		s.SetOnConnect(h)
	}
}

func WithOnDisconnect(h func(session.Sessioner)) ServerOption {
	return func(s *Server) {
		s.SetOnDisconnect(h)
	}
}

func (s *Server) SetOnConnect(h func(session.Sessioner)) {
	s.onConnect = h
}

func (s *Server) SetOnDisconnect(h func(session.Sessioner)) {
	s.onDisconnect = h
}
