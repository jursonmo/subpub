package main

import (
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/jursonmo/subpub/server"
	"github.com/jursonmo/subpub/session"
)

func main() {
	Name := "subpubServer"
	Version := "0.0.1"

	logger := log.With(log.NewStdLogger(os.Stdout),
		"name", Name,
		"version", Version,
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	mylog := log.NewHelper(logger)

	s, err := server.NewServer(
		logger,
		server.WithName("serverExample"),
		server.WithNetwork("tcp4"),
		server.WithAddress("localhost:8000"))

	if err != nil {
		mylog.Errorf("NewServer fail:%v", err)
		return
	}

	onConnect := func(s session.Sessioner) {
		fmt.Printf("connected, id:%s, local:%v, remote:%v\n", s.SessionID(), s.UnderlayConn().LocalAddr(), s.UnderlayConn().RemoteAddr())
	}
	s.SetOnConnect(onConnect)

	onDisonnect := func(s session.Sessioner) {
		fmt.Printf("disconnected, id:%s, local:%v, remote:%v\n", s.SessionID(), s.UnderlayConn().LocalAddr(), s.UnderlayConn().RemoteAddr())
	}
	s.SetOnDisconnect(onDisonnect)

	app := kratos.New(
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Server(s),
	)

	err = app.Run()
	if err != nil {
		mylog.Error(err)
	}
}
