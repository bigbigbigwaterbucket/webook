package main

import (
	"learning_go/webook/pkg/grpcx"
	"learning_go/webook/pkg/mysarama"
)

type App struct {
	//所有需要main函数控制启动、关闭的都在这
	server    *grpcx.Server
	consumers []mysarama.Consumer
}
