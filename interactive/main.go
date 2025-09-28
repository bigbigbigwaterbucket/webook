package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"log"
	"net/http"
)

func main() {
	initViper()
	initPrometheus()
	initLogger()
	app := InitApp()
	for _, consumer := range app.consumers {
		err := consumer.Start()
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("server %#v\n", app.webServer)
	go func() {
		err := app.webServer.Run(":8091")
		log.Println(err)
	}()
	err := app.server.Serve()
	log.Println(err)
}

//func mainV1() {
//	initViper()
//	server := grpc2.NewServer()
//	intrService := InitGRPCService()
//	v1.RegisterInteractiveServiceServer(server, intrService)
//	l, err := net.Listen("tcp", ":8090")
//	if err != nil {
//		panic(err)
//	}
//	err = server.Serve(l)
//	zap.L().Error("微服务interactive错误", zap.Error(err))
//}

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

func initLogger() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	//设置全局logger，全局输出日志的
	//你在你的代码里就可以直接用zap.XXX来记录日志
	zap.ReplaceGlobals(logger)
	//L()获取zap里的全局L（好像是一种类型）logger
	zap.L().Info("日志载入成功")
}

func initPrometheus() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":8085", nil)
		zap.L().Error("err", zap.Error(err))
	}()
}
