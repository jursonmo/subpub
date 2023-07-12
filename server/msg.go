package server

import (
	"errors"

	"github.com/jursonmo/subpub/message"
)

type SessionMsgHandle func(s *Session, msg []byte) error

func (s *Session) RegisterMsgHandler(msgType int, h SessionMsgHandle) {
	s.Lock()
	defer s.Unlock()
	if s.msgHandlers == nil {
		s.msgHandlers = make(map[int]SessionMsgHandle)
	}
	s.msgHandlers[msgType] = h
}

func topicMsgHandle(s *Session, msg []byte) error {
	sm := SubMsg{}
	err := s.Decode(msg, &sm)
	if err != nil {
		s.log().Error(err)
		return err
	}

	if sm.Topic == "" {
		s.log().Error("subscribe topic is empty")
		return errors.New("subscribe topic is empty")
	}

	if sm.Op == message.SubscribeOp {
		s.log().Infof("session subscribe topic:%v", sm.Topic)
		s.SubscribeTopic(sm.Topic)
	} else if sm.Op == message.UnsubscribeOp {
		s.log().Infof("session unsubscribe topic:%v", sm.Topic)
		s.UnsubscribeTopic(sm.Topic)
	}
	return nil
}
