package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	redisv9 "github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"learning_go/webook/internal/config"
	"learning_go/webook/internal/repository"
	"learning_go/webook/internal/repository/cache"
	"learning_go/webook/internal/repository/dao"
	"learning_go/webook/internal/service"
	"learning_go/webook/internal/service/oauth2/wechat"
	"learning_go/webook/internal/service/sms/memoryTest"
	"learning_go/webook/internal/web"
	"learning_go/webook/internal/web/ijwt"
	"learning_go/webook/internal/web/middleware"
	webratelimit "learning_go/webook/pkg/ginx/middleware/ratelimit"
	"learning_go/webook/pkg/ratelimit"
	"strings"
	"time"
)

func main() {

	//u := web.UserHandler{svc: &service.UserServiceI{}}  私有变量没法初始化的，邓明用New方法初始化
	db, err := gorm.Open(mysql.Open(config.Config.MysqlURL))
	redisClient := redisv9.NewClient(&redisv9.Options{
		Addr: config.Config.RedisURL})

	if err != nil {
		//只在初始化过程panic，最小化资源损失
		panic(err) //panic：goroutine直接结束
	}
	err = dao.InitTable(db) //表不存在才会建表
	if err != nil {
		panic(err)
	}
	userDAO := dao.NewUserDAO(db)
	userCache := cache.NewUserCache(redisClient)
	userRepository := repository.NewUserRepository(userDAO, userCache)
	userService := service.NewUserService(userRepository)
	smsSvc := memoryTest.NewMemService()
	codeMemCache := cache.NewMemCodeCache()
	//codeCache := cache.NewCodeCache(redisClient)
	codeRepository := repository.NewCodeRepository(codeMemCache)
	codeService := service.NewCodeService(smsSvc, codeRepository)
	wechatService := wechat.NewWechatService("wx7256bc69ab349c72", "secret")

	//web
	redisJwtHandler := ijwt.NewRedisJwtHandler(redisClient)
	userHandler := web.NewUserHandler(userService, codeService, redisJwtHandler)
	wechatHandler := web.NewOAuth2WechatHandler(wechatService, userService, redisJwtHandler)

	server := gin.Default()

	//middleware也是一种handlerFunc，但是一种AOP的handlerFunc,相当于server的所有的路由都会经过
	//Use函数接收不定个func(*context)类型
	server.Use(func(ctx *gin.Context) {
		//println(ctx.GetHeader("Origin"))
		println("这是第一个 middleware")
	})
	server.Use(func(ctx *gin.Context) {
		println("这是第二个 middleware")
	})

	redisLimiter := ratelimit.NewRedisSlidingWindow(redisClient, 100, time.Second)
	//web服务限流为一分钟100次
	server.Use(webratelimit.NewBuilder(redisLimiter).Build())
	//只会对cors 跨域请求进行限制，postman不会限制？  这里配置的内容就是preflight响应体返回的内容
	server.Use(cors.New(cors.Config{
		//允许的域名最好不要默认所有域名，看前端服务部署在哪个域名端口上
		//AllowOrigins: []string{"http://localhost:3000"}, //字符串切片/数组
		//AllowMethods: []string{"POST", "GET"},
		AllowHeaders:     []string{"authorization", "content-type"},  //允许跨域请求可以额外携带哪些字段
		ExposeHeaders:    []string{"x-jwt-token", "x-refresh-token"}, //允许跨域的前端业务拿到某些字段
		AllowCredentials: true,                                       //是否允许带cookie之类的东西
		//自定义函数判断域名是否允许
		AllowOriginFunc: func(origin string) bool {
			println(origin)
			if strings.HasPrefix(origin, "http://localhost") {
				return true
			}
			return strings.Contains(origin, "yourcompany.com")
		},
		MaxAge: 12 * time.Hour, //preflight请求的有效期，12小时后将再次预检
	}))

	// 使用cookie存储store，即session中的数据，数据将会被加密后存到ssid中
	//store := cookie.NewStore([]byte("secret"))
	//store := memstore.NewStore([]byte("1234567890abcdef1234567890abcdef"), []byte("1234567890abcdef1234567890abcdef"))  //存到服务器内存中

	//第一个参数：最大空闲连接数，面试时问到压力/性能测试确定，第二个参数连接方式tcp，不太可能用udp  第六第七表示身份认证和数据加密
	//数据加密密钥的长度是有限制的，比如要是6字节/16字节
	store, err := redis.NewStore(16, "tcp", config.Config.RedisURL,
		"", "", []byte("1234567890abcdef1234567890abcdef"), []byte("1234567890abcdef1234567890abcdef"))
	if err != nil {
		panic(err)
	}
	//session中间件会设置cookie到浏览器
	server.Use(sessions.Sessions("ssid", store))

	//builder := middleware.LoginMiddlewareBuilder{}
	//server.Use(builder.Build())
	builderJWT := middleware.LoginJWTMiddlewareBuilder{JwtHandler: redisJwtHandler}
	server.Use(builderJWT.AddHPath("/users/login").
		AddHPath("/users/signup").
		AddHPath("/users/login_sms/code/send").
		AddHPath("/users/login_sms").
		AddHPath("/oauth2/wechat/authurl").
		AddHPath("/oauth2/wechat/callback").
		AddHPath("/users/refresh_token").
		Build())

	userHandler.RegisterRouter(server)
	wechatHandler.RegisterRouter(server)

	err = server.Run(":8080")
}
