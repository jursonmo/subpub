package conf

import (
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
)

type Server_HTTP struct {
	Network string `protobuf:"bytes,1,opt,name=network,proto3" json:"network,omitempty"`
	Addr    string `protobuf:"bytes,2,opt,name=addr,proto3" json:"addr,omitempty"`
	//Timeout *durationpb.Duration `protobuf:"bytes,3,opt,name=timeout,proto3" json:"timeout,omitempty"`
	Timeout int
}

type Server struct {
	Http *Server_HTTP `protobuf:"bytes,1,opt,name=http,proto3" json:"http,omitempty"`
	//Grpc *Server_GRPC `protobuf:"bytes,2,opt,name=grpc,proto3" json:"grpc,omitempty"`
}

type Log struct {
	Driver string
	Source string
}

type AppConf struct {
	Name   string `json:"name"`
	Server `json:"server"`
	Log    `json:"log"`
}

func ConfigPrase(configFile string) *AppConf {
	c := config.New(
		config.WithSource(
			file.NewSource(configFile),
		),
	)
	if err := c.Load(); err != nil {
		panic(err)
	}
	v := &AppConf{}
	// Unmarshal the config to struct
	if err := c.Scan(v); err != nil {
		panic(err)
	}
	return v
}
