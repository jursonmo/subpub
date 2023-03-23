package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	_ "github.com/go-kratos/kratos/v2/encoding/json"
	"github.com/go-kratos/kratos/v2/log"
	ws "github.com/gorilla/websocket"
	"github.com/jursonmo/subpub/common"
	"github.com/jursonmo/subpub/message"
)

type Server struct {
	name string
	log  log.Logger
	hlog *log.Helper

	ctx context.Context
	*http.Server
	lis      net.Listener
	tlsConf  *tls.Config
	upgrader *ws.Upgrader

	network string
	address string
	path    string
	timeout time.Duration

	subMutex    sync.RWMutex
	subscribers map[Topic]*Subscribers

	codec encoding.Codec //default json
}

type Subscribers struct {
	subMap map[SessionID]*Subscrber
}

func NewServer(logger log.Logger, opts ...ServerOption) (*Server, error) {
	s := &Server{
		network:     "tcp4",
		address:     "localhost:8080",
		timeout:     time.Second * 10,
		log:         logger,
		codec:       encoding.GetCodec("json"),
		subscribers: make(map[message.Topic]*Subscribers),
	}

	s.upgrader = &ws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	for _, opt := range opts {
		opt(s)
	}

	s.hlog = log.NewHelper(log.With(s.log, "caller", log.Caller(5), "Name", s.name))

	s.Server = &http.Server{
		TLSConfig: s.tlsConf,
	}

	if err := s.listen(); err != nil {
		return nil, err
	}

	http.HandleFunc("/subscribe", s.subscribeHandler)
	http.HandleFunc("/publish", s.publishHandler)
	return s, nil
}

func (s *Server) listen() error {
	if s.lis == nil {
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			return err
		}
		s.lis = lis
	}

	return nil
}

func (s *Server) Start(ctx context.Context) error {
	var err error
	s.ctx = ctx
	s.hlog.Infof("listen on:%s,%s", s.network, s.address)
	if s.tlsConf != nil {
		err = s.ServeTLS(s.lis, "", "")
	} else {
		err = s.Serve(s.lis)
	}

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[websocket] server stopping")
	return s.Shutdown(ctx)
}

func info(conn *ws.Conn) string {
	return fmt.Sprintf("l:%v<->r:%v", conn.LocalAddr(), conn.RemoteAddr())
}

func (s *Server) subscribeHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := s.upgrader.Upgrade(res, req, nil)
	if err != nil {
		s.hlog.Errorf("[websocket] upgrade exception:", err)
		return
	}
	defer conn.Close()
	s.hlog.Infof("new conn:%v", info(conn))

	session := NewSession(conn, s)
	session.Start(s.ctx)
	session.wait()
	s.hlog.Errorf("subscribeHandler:%v over---", info(conn))
}

func (s *Server) publishHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := s.upgrader.Upgrade(res, req, nil)
	if err != nil {
		s.hlog.Errorf("[websocket] upgrade exception:", err)
		return
	}
	defer conn.Close()

	//conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	common.SetReadDeadline(conn, true, time.Second*8)
	for {
		t, msg, err := conn.ReadMessage()
		if err != nil {
			s.hlog.Error(err)
			return
		}
		conn.SetReadDeadline(time.Time{})
		if t != ws.BinaryMessage {
			s.hlog.Errorf("unsuport msg type:%v, only support BinaryMessage:%d", t, ws.BinaryMessage)
			return
		}
		pubMsg := PubMsg{}
		err = s.codec.Unmarshal(msg, &pubMsg)
		if err != nil {
			s.hlog.Error(err)
			continue
		}
		s.hlog.Debugf("%v publish topic:%s", info(conn), pubMsg.Topic)
		s.public(pubMsg.Topic, pubMsg.Body)
	}
}

func (s *Server) public(topic Topic, data []byte) {
	s.subMutex.RLock()
	defer s.subMutex.RUnlock()
	subers, ok := s.subscribers[topic]
	if !ok {
		return
	}
	for _, sub := range subers.subMap {
		sub.put(PubMsg{Topic: topic, Body: data})
	}
}

func (c *Session) Conn() *ws.Conn {
	return c.conn
}

func (c *Session) SessionID() SessionID {
	return c.id
}

func (s *Server) AddSubscriber(sub *Subscrber, topic Topic) {
	s.subMutex.Lock()
	defer s.subMutex.Unlock()
	subscribers, ok := s.subscribers[topic]
	if !ok {
		subMap := make(map[SessionID]*Subscrber)
		subMap[sub.id] = sub

		s.subscribers[topic] = &Subscribers{subMap: subMap}
		return
	}
	subscribers.subMap[sub.id] = sub
}

func (s *Server) RemoveSubscriber(sub *Subscrber, topic Topic) {
	s.subMutex.Lock()
	defer s.subMutex.Unlock()
	subscribers, ok := s.subscribers[topic]
	if !ok {
		return
	}
	delete(subscribers.subMap, sub.id)
}
