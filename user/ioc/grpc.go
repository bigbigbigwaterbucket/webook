package ioc

import (
	"learning_go/webook/api/proto/gen/user/userv1"
	"learning_go/webook/pkg/grpcx"
	"learning_go/webook/user/grpc"
	"learning_go/webook/user/service"

	"github.com/spf13/viper"
	grpc2 "google.golang.org/grpc"
)

func InitGRPCXServer(svc service.UserService) *grpcx.Server {
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
	grpcSvc := grpc.NewUserServiceServer(svc)
	userv1.RegisterUserServiceServer(server, grpcSvc)
	return &grpcx.Server{Server: server, Name: "user", Port: config1.Port, EtcdAddrs: config1.EtcdAddrs}
}
