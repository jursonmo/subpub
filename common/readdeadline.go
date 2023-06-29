package common

import (
	"time"

	"github.com/gorilla/websocket"
)

func SetReadDeadline(conn *websocket.Conn, isServer bool, readDeadline time.Duration) {
	//fix:先设置一个deadline, 如果没有收到ping or pong 自动超时, 收到ping or pong 后 会更新deadline
	conn.SetReadDeadline(time.Now().Add(readDeadline)) //如果这里不设置deadline，可能一直收不到ping,那么永远都无法超时

	wrapHanlderWithDeadline := func(handler func(string) error, readDeadline time.Duration) func(string) error {
		return func(s string) error {
			conn.SetReadDeadline(time.Now().Add(readDeadline))
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
}
