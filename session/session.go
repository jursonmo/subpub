package session

import "net"

type Sessioner interface {
	SessionID() string
	UnderlayConn() net.Conn
	//Endpoints() []string
}
