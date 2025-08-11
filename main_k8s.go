package main

import (
	"github.com/gin-gonic/gin"
)

func main2() {
	server := gin.Default()
	server.GET("/hello", func(ctx *gin.Context) {
		ctx.String(200, "你好")
	})
	server.Run(":8080")
}
