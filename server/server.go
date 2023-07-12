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

	ctx    context.Context
	cancel context.CancelFunc
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

	pathHandlers map[string]WsHandler //key:path, value:handler
}

type WsHandler func(s *Server, w http.ResponseWriter, r *http.Request)

type Subscribers struct {
	subMap map[SessionID]*Subscrber
}

func NewServer(logger log.Logger, opts ...ServerOption) (*Server, error) {
	s := &Server{
		network:      "tcp4",
		address:      "localhost:8080",
		timeout:      time.Second * 10,
		log:          logger,
		codec:        encoding.GetCodec("json"),
		subscribers:  make(map[message.Topic]*Subscribers),
		pathHandlers: make(map[string]WsHandler),
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

	//option path handler 注册到ws HandleFunc
	for path, handler := range s.pathHandlers {
		http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			handler(s, w, r)
		})
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
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.hlog.Infof("listen on:%s,%s", s.network, s.address)
	if s.tlsConf != nil {
		// 其实就是tlsListener := tls.NewListener(ln, tlsConfig) + s.Serve(tlsListener)
		// 如果没有给出证书路径，就默认用http server 的 TLSConfig 配置
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
	if s.cancel != nil {
		s.cancel() //make all session quit
	}
	return s.Shutdown(ctx)
}

func info(conn *ws.Conn) string {
	return fmt.Sprintf("l:%v<->r:%v", conn.LocalAddr(), conn.RemoteAddr())
}

func (s *Server) subscribeHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := s.upgrader.Upgrade(res, req, nil)
	if err != nil {
		s.hlog.Error("[websocket] upgrade exception:", err)
		return
	}
	defer conn.Close()
	s.hlog.Infof("new conn:%v", info(conn))

	session := NewSession(conn, s, SessionTimeout(s.timeout))
	session.RegisterMsgHandler(ws.BinaryMessage, topicMsgHandle)
	session.Start(s.ctx)
	err = session.wait()
	s.hlog.Errorf("subscribeHandler over:%v, err:%v", info(conn), err)
}

func (s *Server) publishHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := s.upgrader.Upgrade(res, req, nil)
	if err != nil {
		s.hlog.Error("[websocket] upgrade exception:", err)
		return
	}
	defer conn.Close()

	//conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	err = common.SetReadDeadline(conn, true, time.Second*8)
	if err != nil {
		s.hlog.Error(err)
		return
	}
	for {
		t, msg, err := conn.ReadMessage()
		if err != nil {
			s.hlog.Error(err)
			return
		}
		//conn.SetReadDeadline(time.Time{}) //不需要重置为不超时，common.SetReadDeadline会不断重置超时时间
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
		n := s.public(pubMsg.Topic, pubMsg.Body)
		if n == 0 {
			s.hlog.Warnf("there is no subscribers on this topic:%v", pubMsg.Topic)
		}
	}
}

func (s *Server) public(topic Topic, data []byte) int {
	s.subMutex.RLock()
	defer s.subMutex.RUnlock()
	subers, ok := s.subscribers[topic]
	if !ok {
		return 0
	}
	for _, sub := range subers.subMap {
		sub.put(PubMsg{Topic: topic, Body: data})
	}
	return len(subers.subMap)
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
	//there is no subscribers on this topic?
	if len(subscribers.subMap) == 0 {
		delete(s.subscribers, topic)
	}
}
