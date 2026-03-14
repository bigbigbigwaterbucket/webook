package ioc

import (
	"learning_go/webook/api/proto/gen/comment/comv1"
	grpc2 "learning_go/webook/comment/grpc"
	"learning_go/webook/comment/service"

	"learning_go/webook/pkg/grpcx"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// 类似于web层，依赖service，要自己启动一个rpc server，然后把自己的server注册到rpc server
func InitGRPCXServer(svc service.CommentService) *grpcx.Server {
	type Config struct {
		Port      int      `yaml:"port"`
		EtcdAddrs []string `yaml:"etcdAddrs"`
	}
	var config1 Config
	err := viper.UnmarshalKey("grpc.server", &config1)
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer()
	server := grpc2.NewCommentServiceServer(svc)
	comv1.RegisterCommentServiceServer(grpcServer, server)
	return &grpcx.Server{Server: grpcServer, Name: "comment", Port: config1.Port, EtcdAddrs: config1.EtcdAddrs}
}
