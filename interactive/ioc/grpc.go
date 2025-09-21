package ioc

import (
	"github.com/spf13/viper"
	grpc2 "google.golang.org/grpc"
	v1 "learning_go/webook/api/proto/gen/interactive/intrv1"
	"learning_go/webook/interactive/grpc"
	"learning_go/webook/pkg/grpcx"
)

func InitGRPCXServer(svc *grpc.InteractiveServiceServer) *grpcx.Server {
	type Config struct {
		Addr string `yaml:"addr"`
	}
	var config1 Config
	err := viper.UnmarshalKey("grpc.server", &config1)
	if err != nil {
		panic(err)
	}
	server := grpc2.NewServer()
	v1.RegisterInteractiveServiceServer(server, svc)
	return &grpcx.Server{Addr: config1.Addr, Server: server}
}
