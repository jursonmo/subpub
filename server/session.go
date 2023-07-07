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
	"golang.org/x/sync/errgroup"
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
	//when session closed, need to unsubscribe all topics that already subscribed
	topics  []Topic
	id      SessionID
	conn    *ws.Conn
	server  *Server
	timeout time.Duration //heartbeat timeout and close session
	send    chan []byte
	codec   encoding.Codec
	done    chan struct{}
	closed  bool
	eg      *errgroup.Group
}
type SessionOpt func(*Session)

func SessionTimeout(d time.Duration) SessionOpt {
	return func(s *Session) {
		s.timeout = d
	}
}

func NewSession(conn *ws.Conn, s *Server, opts ...SessionOpt) *Session {
	u1, err := uuid.NewUUID()
	if err != nil {
		s.hlog.Error(err)
	}
	session := &Session{
		id:     SessionID(u1.String()),
		conn:   conn,
		send:   make(chan []byte, channelBufSize),
		server: s,
		done:   make(chan struct{}),
	}
	for _, opt := range opts {
		opt(session)
	}
	return session
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
		s.log().Errorf("%v send queue is full", s)
	}
}

func (s *Session) wait() error {
	//<-s.done
	return s.eg.Wait()
}

func (s *Session) close() {
	s.Lock()
	defer s.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	//close(s.done) //instead of errgroup

	for _, topic := range s.topics {
		s.server.RemoveSubscriber(s, topic)
	}
	s.conn.Close()

	s.log().Errorf("%v close ok", s)
}

func (s *Session) Start(ctx context.Context) {
	// go s.writeLoop(ctx)
	// go s.readLoop(ctx)
	s.eg, ctx = errgroup.WithContext(ctx)
	s.eg.Go(func() error {
		return s.writeLoop(ctx)
	})
	s.eg.Go(func() error {
		return s.readLoop(ctx)
	})
}

func (s *Session) SubscribeTopic(topic Topic) {
	exist := false
	s.Lock()
	for _, t := range s.topics {
		if t == topic {
			exist = true
		}
	}
	if !exist {
		s.topics = append(s.topics, topic)
	}
	s.Unlock()

	if !exist {
		s.server.AddSubscriber(s, topic)
	}
}

func (s *Session) UnsubscribeTopic(topic Topic) {
	removeIndex := 0
	exist := false
	s.Lock()
	for i, t := range s.topics {
		if t == topic {
			exist = true
			removeIndex = i
			break
		}
	}
	if exist {
		s.topics = append(s.topics[0:removeIndex], s.topics[removeIndex+1:]...)
	}
	s.Unlock()

	if exist {
		s.server.RemoveSubscriber(s, topic)
	}
}

func (s *Session) readLoop(ctx context.Context) error {
	defer s.close()
	defer s.log().Errorf("Session:%v readLoop quit", info(s.conn))

	//conn.SetReadDeadline(time.Second * 2)
	if s.timeout == 0 {
		panic("seesion timeout eq 0")
	}
	err := common.SetReadDeadline(s.conn, true, s.timeout)
	if err != nil {
		return err
	}
	for {
		messageType, msg, err := s.conn.ReadMessage()
		if err != nil {
			s.log().Errorf("Error during message reading:%v", err)
			return err
		}
		if messageType != ws.BinaryMessage {
			s.log().Errorf("subscribe msg don't support messageType:%v, but only BinaryMessage, close:%v",
				messageType, info(s.conn))
			continue
		}
		sm := SubMsg{}
		err = s.Decode(msg, &sm)
		if err != nil {
			s.log().Error(err)
			continue
		}

		if sm.Topic == "" {
			s.log().Error("subscribe topic is empty")
			continue
		}

		if sm.Op == message.SubscribeOp {
			s.log().Infof("session subscribe topic:%v", sm.Topic)
			s.SubscribeTopic(sm.Topic)
		} else if sm.Op == message.UnsubscribeOp {
			s.log().Infof("session unsubscribe topic:%v", sm.Topic)
			s.UnsubscribeTopic(sm.Topic)
		}
	}
}

func (s *Session) writeLoop(ctx context.Context) error {
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
		// case <-s.done:
		// 	return
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
