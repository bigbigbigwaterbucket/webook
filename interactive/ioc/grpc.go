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
		Port      int      `yaml:"port"`
		EtcdAddrs []string `yaml:"etcdAddrs"`
	}
	var config1 Config
	err := viper.UnmarshalKey("grpc.server", &config1)
	if err != nil {
		panic(err)
	}
	server := grpc2.NewServer()
	v1.RegisterInteractiveServiceServer(server, svc)
	return &grpcx.Server{Name: "interactive", EtcdAddrs: config1.EtcdAddrs, Port: config1.Port, Server: server}
}

//type Server struct {
//	*grpc.Server
//	Name            string
//	EtcdAddrs       []string
//	port            int
//	client          *etcdv3.Client
//	kaCancel        func()
//	endPointManager endpoints.Manager
//	key             string
//}
