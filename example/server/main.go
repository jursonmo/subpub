package main

import (
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

func appStart(configFile string) error {
	appConf := conf.ConfigPrase(configFile)
	fmt.Printf("config: %+v\n", appConf)

	mylog := log.NewHelper(log.With(log.GetLogger(), "caller", log.Caller(4)))

	s, err := server.NewServer(
		log.GetLogger(),
		server.WithNetwork(appConf.Http.Network),
		server.WithAddress(appConf.Http.Addr))

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