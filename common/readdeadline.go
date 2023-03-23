package common

import (
	"time"

	"github.com/gorilla/websocket"
)

func SetReadDeadline(conn *websocket.Conn, isServer bool, readDeadline time.Duration) {
	wrapHanlder := func(handler func(string) error, readDeadline time.Duration) func(string) error {
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
		conn.SetPingHandler(wrapHanlder(handler, readDeadline))
	} else {
		handler = conn.PongHandler()
		conn.SetPongHandler(wrapHanlder(handler, readDeadline))
	}
}
