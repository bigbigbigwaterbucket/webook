package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	grpc2 "google.golang.org/grpc"
	v1 "learning_go/webook/api/proto/gen/interactive/v1"
	"net"
)

func main() {
	initViper()
	server := grpc2.NewServer()
	intrService := InitGRPCService()
	v1.RegisterInteractiveServiceServer(server, intrService)
	l, err := net.Listen("tcp", ":8090")
	if err != nil {
		panic(err)
	}
	err = server.Serve(l)
	zap.L().Error("微服务interactive错误", zap.Error(err))
}

func initViper() {
	//从程序运行参数program arguments那读取配置参数,value是默认值，返回的是地址，说明后续pflag还会修改指针指向的值
	cp := pflag.String("config", "./webook/interactive/config", "指定配置文件路径")
	pflag.Parse() //调用该函数对cp赋值
	println(*cp)
	//路径、文件名、文件类型配置
	viper.AddConfigPath(*cp)
	viper.SetConfigName("dev")
	viper.SetConfigType("yaml")
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		println("本地配置文件已经改变")
	})
	//设置默认值，也可以放在结构体初始化那里作为默认值
	viper.SetDefault("mysql.dsn", "root:root@tcp(webook-mysql:3308)/webook")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}
