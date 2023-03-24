package main

import (
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/jursonmo/subpub/server"
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
		server.WithAddress("localhost:8080"))

	if err != nil {
		mylog.Errorf("NewServer fail:%v", err)
		return
	}

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
