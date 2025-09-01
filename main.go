package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	redisv9 "github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"learning_go/webook/internal/config"
	"learning_go/webook/internal/repository"
	"learning_go/webook/internal/repository/article"
	"learning_go/webook/internal/repository/cache"
	"learning_go/webook/internal/repository/dao"
	article2 "learning_go/webook/internal/repository/dao/article"
	"learning_go/webook/internal/service"
	"learning_go/webook/internal/service/oauth2/wechat"
	"learning_go/webook/internal/service/sms/sms_implementation/memoryTest"
	"learning_go/webook/internal/web"
	"learning_go/webook/internal/web/ijwt"
	"learning_go/webook/internal/web/middleware"
	"learning_go/webook/pkg/ginx/middleware/logger"
	webratelimit "learning_go/webook/pkg/ginx/middleware/ratelimit"
	"learning_go/webook/pkg/ratelimit"
	"strings"
	"time"
)

func initViperRemote() {
	//引入远程配置etcd前，别忘了加载viper的remote包
	viper.SetConfigName("dev")
	viper.SetConfigType("yaml")
	//两段配置：先读配置文件的信息，然后链接远程配置中心
	err := viper.AddRemoteProvider("etcd3", "127.0.0.1:12379", "/webook")
	if err != nil {
		panic(err)
	}
	err = viper.WatchRemoteConfig()
	if err != nil {
		panic(err)
	}
	//没有用，不支持监听远程
	viper.OnConfigChange(func(in fsnotify.Event) {
		println("文件已经修改")
	})
	err = viper.ReadRemoteConfig()
	if err != nil {
		panic(err)
	}
}

func initViper() {
	//从程序运行参数那读取配置参数,value是默认值，返回的是地址，说明后续pflag还会修改指针指向的值
	cp := pflag.String("config", "./webook/internal/config", "指定配置文件路径")
	pflag.Parse() //调用该函数对cp赋值
	println(*cp)
	//路径、文件名、文件类型配置
	viper.AddConfigPath("./webook/internal/config")
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

func main() {
	type Config struct {
		DSN string `yaml:"dsn"` //反序列化读进来的，注意大写
	}
	initLogger()
	initViper()
	//initViperRemote()
	var config1 Config
	//远程链接etcd时，viper不支持对yaml文件分隔符.的解析！！！
	err := viper.UnmarshalKey("mysql", &config1)
	if err != nil {
		panic(err)
	}
	fmt.Println(config1.DSN)

	//u := web.UserHandler{svc: &service.UserServiceI{}}  私有变量没法初始化的，邓明用New方法初始化
	db, err := gorm.Open(mysql.Open(config.Config.MysqlURL), &gorm.Config{Logger: glogger.New(gormLoggerFunc(zap.L().Debug),
		glogger.Config{
			LogLevel:                  glogger.Info,
			SlowThreshold:             time.Millisecond * 10, //慢启动阈值，记录哪些sql执行时间慢于10ms
			IgnoreRecordNotFoundError: true,                  //是否忽略record没找到错误，这在某些情况下是比较常见的
			// ParameterizedQueries:      true,将插入的数据屏蔽掉，安全考虑
		})})
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
	//codeMemCache := cache.NewMemCodeCache()
	codeCache := cache.NewCodeCache(redisClient)
	codeDao := dao.NewCodeDaoI(db)
	codeRepository := repository.NewCodeRepository(codeCache, codeDao)
	codeService := service.NewCodeService(smsSvc, codeRepository)
	wechatService := wechat.NewWechatService("wx7256bc69ab349c72", "secret")
	articleDao := article2.NewGormArticleDao(db)
	articleCache := cache.NewRedisArticleCache(redisClient)
	articleRepository := article.NewCachedArticleRepository(articleDao, articleCache, userRepository)
	articleService := service.NewArticleServiceI(articleRepository)

	//web
	redisJwtHandler := ijwt.NewRedisJwtHandler(redisClient)
	userHandler := web.NewUserHandler(userService, codeService, redisJwtHandler)
	wechatHandler := web.NewOAuth2WechatHandler(wechatService, userService, redisJwtHandler)
	articleHandler := web.NewArticleHandler(articleService)

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
	server.Use(logger.NewLoggerBuilder(func(ctx *gin.Context, log *logger.AccessLog) {
		zap.L().Debug("请求与响应信息", zap.Any("请求与响应", log))
	}).AllowRespBody().AllowReqBody().Build())

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
	articleHandler.RegisterRouter(server)

	err = server.Run(":8080")
}

type gormLoggerFunc func(msg string, fileds ...zap.Field)

func (g gormLoggerFunc) Printf(msg string, args ...interface{}) {
	g(msg, zap.Any("gormArgs", args))
}
