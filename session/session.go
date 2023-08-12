package session

import (
	"errors"
	"net"

	"github.com/jursonmo/subpub/message"
)

type Sessioner interface {
	SessionID() string
	UnderlayConn() net.Conn
	//Endpoints() []string

	WriteMessage(d []byte) error
	Subscribe(topic string) (chan []byte, error)
	SubscribeWithHandler(topic string, handler message.TopicMsgHandler) error
	Publish(topic string, content []byte) error
}

type BaseSession struct{}

var ErrNonImplement = errors.New("non implement")

func (bs *BaseSession) WriteMessage(d []byte) error {
	return ErrNonImplement
}
func (bs *BaseSession) Subscribe(topic string) (chan []byte, error) {
	return nil, ErrNonImplement
}
func (bs *BaseSession) Publish(topic string, content []byte) error {
	return ErrNonImplement
}
func (bs *BaseSession) SubscribeWithHandler(topic string, handler message.TopicMsgHandler) error {
	return ErrNonImplement
}
