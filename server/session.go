package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
	"github.com/jursonmo/subpub/common"
	"github.com/jursonmo/subpub/message"
)

type Topic = message.Topic
type SubMsg = message.SubMsg
type PubMsg = message.PubMsg
type SessionID string

var channelBufSize = 128

//subscrber as session
type Subscrber = Session
type Session struct {
	sync.Mutex
	topics []Topic
	id     SessionID
	conn   *ws.Conn
	server *Server
	send   chan []byte
	codec  encoding.Codec
	done   chan struct{}
	closed bool
}

func NewSession(conn *ws.Conn, s *Server) *Session {
	u1, err := uuid.NewUUID()
	if err != nil {
		s.hlog.Error(err)
	}
	return &Session{
		id:     SessionID(u1.String()),
		conn:   conn,
		send:   make(chan []byte, channelBufSize),
		server: s,
		done:   make(chan struct{}),
	}
}

func (s *Session) String() string {
	return fmt.Sprintf("subscriber:%s", info(s.conn))
}

func (s *Session) log() *log.Helper {
	return s.server.hlog
}
func (s *Session) Codec() encoding.Codec {
	codec := s.codec
	if codec == nil {
		codec = s.server.codec
	}
	return codec
}

func (s *Session) Decode(data []byte, v interface{}) error {
	return s.Codec().Unmarshal(data, v)
}

func (s *Session) Encode(v interface{}) ([]byte, error) {
	//sm.Encode = s.Codec().Name()
	return s.Codec().Marshal(v)
}

func (s *Session) put(m PubMsg) {
	data, err := s.Encode(&m)
	if err != nil {
		s.log().Error(err)
		return
	}
	select {
	case s.send <- data:
	default:
		s.log().Error("%v send queue is full", s)
	}
}

func (s *Session) wait() {
	<-s.done
}

func (s *Session) close() {
	s.Lock()
	defer s.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	close(s.done)

	for _, topic := range s.topics {
		s.server.RemoveSubscriber(s, topic)
	}
	s.conn.Close()

	s.log().Errorf("%v close ok", s)
}
func (s *Session) Start(ctx context.Context) {
	go s.writeLoop(ctx)
	go s.readLoop(ctx)
}

func (s *Session) AddTopic(topic Topic) {
	exist := false
	s.Lock()
	for _, t := range s.topics {
		if t == topic {
			exist = true
		}
	}
	if !exist {
		s.topics = append(s.topics, Topic(topic))
	}
	s.Unlock()

	if !exist {
		s.server.AddSubscriber(s, Topic(topic))
	}
}

func (s *Session) readLoop(ctx context.Context) {
	defer s.close()
	defer s.log().Errorf("Session:%v readLoop quit", info(s.conn))

	//conn.SetReadDeadline(time.Second * 2)
	common.SetReadDeadline(s.conn, true, 8*time.Second)
	for {
		messageType, message, err := s.conn.ReadMessage()
		if err != nil {
			s.log().Errorf("Error during message reading:%v", err)
			return
		}
		if messageType != ws.BinaryMessage {
			s.log().Errorf("subscribe msg don't support messageType:%v, but only BinaryMessage, close:%v",
				messageType, info(s.conn))
			continue
		}
		sm := SubMsg{}
		err = s.Decode(message, &sm)
		if err != nil {
			s.log().Error(err)
			continue
		}

		if sm.Topic == "" {
			s.log().Error("subscribe topic is empty")
			continue
		}
		s.AddTopic(sm.Topic)
	}
}

func (s *Session) writeLoop(ctx context.Context) {
	var err error
	defer s.close()
	defer s.log().Errorf("Session:%v writeLoop quit", info(s.conn))
	for {
		select {
		case data := <-s.send:
			err = s.conn.WriteMessage(ws.BinaryMessage, data)
			if err != nil {
				s.log().Error(err)
			}
		case <-s.done:
			return
		case <-ctx.Done():
			return
		}
	}
}
