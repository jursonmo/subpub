package main

import (
	"crypto/tls"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/jursonmo/subpub/cmd"
	"github.com/jursonmo/subpub/conf"
	"github.com/jursonmo/subpub/server"
	"github.com/spf13/cobra"
)

var (
	configPath string
	Version    string = "unset by build"
)

func init() {
	cmd.Version = Version
	RootCmd.AddCommand(cmd.VersionCmd)
	RootCmd.Flags().StringVarP(&configPath, "config", "c", "./config.yaml", "config file path")
}

var RootCmd = &cobra.Command{
	Use:     "subscribe and publish topic",
	Short:   "short: subpub program",
	Long:    "long: ",
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Inside rootCmd PersistentPreRun with args: %v\n", args)
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Inside rootCmd PreRun with args: %v\n", args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("rootCmd Run\n")
		fmt.Printf("args:%v, configPath:%s\n", args, configPath)
		appStart(configPath)
	},
}

func getTlsConfig(crt, key string) *tls.Config {
	cert, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		return nil
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}
}

func appStart(configFile string) error {
	appConf := conf.ConfigPrase(configFile)
	fmt.Printf("config: %+v\n", appConf)

	InitLogger(appConf.Log.Source)
	mylog := log.NewHelper(log.With(log.GetLogger(), "caller", log.Caller(4)))

	tlsConf := getTlsConfig("../cert/server.crt", "../cert/server.key")

	s, err := server.NewServer(
		log.GetLogger(),
		server.WithName(appConf.Name),
		server.WithNetwork(appConf.Websocket.Network),
		server.WithAddress(appConf.Websocket.Addr),
		server.WithTLSConfig(tlsConf))

	if err != nil {
		mylog.Errorf("NewServer fail:%v", err)
		return err
	}

	myapp := kratos.New(
		kratos.Name(appConf.Name),
		kratos.Version(Version),
		kratos.Server(s),
	)

	err = myapp.Run()
	if err != nil {
		mylog.Error(err)
	}
	return err
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func InitLogger(logPath string) {
	//zaplog := myzap.NewLogger(zap.NewExample())
	//logger := log.With(zaplog)
	f, err := os.Create(logPath)
	if err != nil {
		panic(err)
	}

	logger := log.With(log.NewStdLogger(f),
		"ts", log.DefaultTimestamp,
		// 	"caller", log.Caller(4), //log.DefaultCaller,
		// 	"service.id", id,
		// 	"service.name", Name,
		// 	"service.version", Version,
		// 	"trace_id", tracing.TraceID(),
		// 	"span_id", tracing.SpanID(),
	)
	_ = logger
	log.SetLogger(logger)

	//test log
	// glogger := log.GetLogger()
	// slog := log.With(glogger, "service.name", "helloworld")
	// slog.Log(log.LevelInfo, "testkey", "testvalue")
}
