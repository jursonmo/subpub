package common

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	SubscriberPath = "/subscribe"
	PublisherPath  = "/publish"
)

func SetReadDeadline(conn *websocket.Conn, isServer bool, readDeadline time.Duration) error {
	//fix:先设置一个deadline, 如果没有收到ping or pong 自动超时, 收到ping or pong 后 会更新deadline
	err := conn.SetReadDeadline(time.Now().Add(readDeadline)) //如果这里不设置deadline，可能一直收不到ping,那么永远都无法超时
	if err != nil {
		return err
	}
	wrapHanlderWithDeadline := func(handler func(string) error, readDeadline time.Duration) func(string) error {
		return func(s string) error {
			//每次收到ping or pong 心跳都要重置deadline，否则就超时
			_ = conn.SetReadDeadline(time.Now().Add(readDeadline)) // "_ =" is for golangci-lint errcheck
			if handler != nil {
				return handler(s)
			}
			return nil
		}
	}

	var handler func(string) error
	if isServer {
		handler = conn.PingHandler()
		conn.SetPingHandler(wrapHanlderWithDeadline(handler, readDeadline))
	} else {
		handler = conn.PongHandler()
		conn.SetPongHandler(wrapHanlderWithDeadline(handler, readDeadline))
	}
	return nil
}
