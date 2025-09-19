package main

import (
	"context"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/fsnotify/fsnotify"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	redisv9 "github.com/redis/go-redis/v9"
	cron2 "github.com/robfig/cron/v3"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	gormPrometheus "gorm.io/plugin/prometheus"
	articleEvent "learning_go/webook/interactive/events"
	repository2 "learning_go/webook/interactive/repository"
	cache2 "learning_go/webook/interactive/repository/cache"
	dao2 "learning_go/webook/interactive/repository/dao"
	service2 "learning_go/webook/interactive/service"
	"learning_go/webook/internal/config"
	job2 "learning_go/webook/internal/job"
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
	"learning_go/webook/pkg/ginx"
	"learning_go/webook/pkg/ginx/middleware/metrics"
	webratelimit "learning_go/webook/pkg/ginx/middleware/ratelimit"
	"learning_go/webook/pkg/ratelimit"
	"learning_go/webook/pkg/redisx"
	"net/http"
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

func initPrometheus() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":8085", nil)
		zap.L().Error("err", zap.Error(err))
	}()
}

func initOpenTelemetry() func(ctx context.Context) {
	//当前服务的资源
	res, err := newResource("webook", "v0.0.1")
	if err != nil {
		panic(err)
	}
	//上下文配置
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)
	//追踪trace，使用zipkin
	tp, err := newTraceProvider(res)
	if err != nil {
		panic(err)
	}
	otel.SetTracerProvider(tp)
	//退出追踪，闭包函数
	return func(ctx context.Context) {
		tp.Shutdown(ctx)
	}
}

func main() {
	type Config struct {
		DSN string `yaml:"dsn"` //反序列化读进来的，注意大写
	}
	initLogger()
	initViper()
	closeFunc := initOpenTelemetry()
	initPrometheus()
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
	interactiveDao := dao2.NewGORMInteractiveDao(db)
	interactiveCache := cache2.NewRedisInteractiveCache(redisClient)
	interactiveRepository := repository2.NewCachedInteractiveRepository(time.Minute*10, interactiveDao, interactiveCache)
	interactiveService := service2.NewInteractiveServiceI(interactiveRepository)
	redisRankingCache := cache.NewRedisRankingCache(redisClient, "ranking")
	localRankingCache := cache.NewLocalRankingCache(time.Minute * 10) //这里三数据的本地缓存过期时间对齐redis
	rankingRepo := repository.NewOnlyCachedRankingRepository(redisRankingCache, localRankingCache)
	rankingService := service.NewRankingServiceI(articleRepository, interactiveRepository, rankingRepo)

	//consumer
	var address = []string{"localhost:9094"}
	saramaConfig := sarama.NewConfig()
	client, err := sarama.NewClient(address, saramaConfig)
	if err != nil {
		panic(err)
	}
	interactiveConsumer := articleEvent.NewInteractiveReadEventBatchConsumer(client, interactiveRepository)
	err = interactiveConsumer.Start()
	if err != nil {
		panic(err)
	}
	//web
	redisJwtHandler := ijwt.NewRedisJwtHandler(redisClient)
	userHandler := web.NewUserHandler(userService, codeService, redisJwtHandler)
	wechatHandler := web.NewOAuth2WechatHandler(wechatService, userService, redisJwtHandler)
	articleHandler := web.NewArticleHandler(articleService, interactiveService)
	// 用来测试prometheus的观测功能的接口，方便给wrk压测
	observeHandler := web.NewObservabilityHandler()

	server := gin.Default()

	//middleware也是一种handlerFunc，但是一种AOP的handlerFunc,相当于server的所有的路由都会经过
	//Use函数接收不定个func(*context)类型

	//server.Use(func(ctx *gin.Context) {
	//	//println(ctx.GetHeader("Origin"))
	//	println("这是第一个 middleware")
	//})
	//server.Use(func(ctx *gin.Context) {
	//	println("这是第二个 middleware")
	//})

	//先暂时注释掉，否则prometheus会请求大量信息导致输出大量日志
	//server.Use(logger.NewLoggerBuilder(func(ctx *gin.Context, log *logger.AccessLog) {
	//	zap.L().Debug("请求与响应信息", zap.Any("请求与响应", log))
	//}).AllowRespBody().AllowReqBody().Build())

	redisClient.AddHook(redisx.NewPrometheusHook(prometheus.SummaryOpts{
		Namespace: "waterbucket", Subsystem: "webook",
		Name: "redis", Help: "redis执行时间检测与是否命中检测",
		Objectives: map[float64]float64{
			0.5:  0.01,
			0.75: 0.01,
			0.9:  0.005,
			0.99: 0.001,
		},
	}))

	ginx.InitOpt(prometheus.CounterOpts{
		Namespace: "waterbucket", Subsystem: "webook",
		Name: "gin_web", Help: "在wrapper封装的路由函数中统计http的业务错误码",
	})

	server.Use((&metrics.MiddleWareBuilder{Namespace: "waterbucket", Subsystem: "webook", //这里不要用连字符，会报错
		Name: "gin_web", Help: "统计gin的http接口响应时间", InstanceID: "localhost:8080"}).Builder())

	//这里提供的接口是去检测sql的一些指标
	err = db.Use(gormPrometheus.New(gormPrometheus.Config{
		DBName:          "webook",
		RefreshInterval: 15,    //拉取间隔
		StartServer:     false, //已经开启prometheus的handler了，不需要重新开启服务
		MetricsCollector: []gormPrometheus.MetricsCollector{
			&gormPrometheus.MySQL{
				VariableNames: []string{"thread_running"}, //不知道啥意思，大明也不懂
			},
		},
	}))
	if err != nil {
		panic(err)
	}

	sqlVector := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "waterbucket",
		Subsystem: "webook",
		Name:      "gorm_query_time",
		Objectives: map[float64]float64{
			0.5:  0.01,
			0.75: 0.01,
			0.9:  0.005,
			0.99: 0.001,
		},
		//table方便定位不同表的查询用时
		//type方便定位是insert语句还是其他语句
		//如果是join语句，可以按主表A join B的A来，也可以合在一起传
	}, []string{"type", "table"})
	//定义好指标后别忘记注册！
	prometheus.MustRegister(sqlVector)

	//callbacks集中使用gorm的callback机制注册prometheus summary分位数指标
	callbacks := Callbacks{SqlVector: sqlVector}
	callbacks.initCallbacks(db)

	redisLimiter := ratelimit.NewRedisSlidingWindow(redisClient, 1000, time.Second)
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
		AddHPath("/test/metric").
		AddHPath("/articles/pub/liketop").
		Build())

	//注册openTelemetry，监控traces
	server.Use(otelgin.Middleware("webook"))
	userHandler.RegisterRouter(server)
	wechatHandler.RegisterRouter(server)
	articleHandler.RegisterRouter(server)
	observeHandler.RegisterRoutes(server)

	//注册定时任务
	expr := cron2.New()
	//一次分批查询的总时长为60s，取决于近七天的数据量
	job := job2.NewPrometheusJobBuilder().Build(job2.NewRankingJob(redisClient, rankingService, time.Minute))
	//每三分钟一次
	_, err = expr.AddJob("0 */3 * * * ?", job)
	if err != nil {
		panic(err)
	}
	expr.Start()

	//放在注册路由后面的midleware不会生效！！！
	server.Use(func(ctx *gin.Context) {
		println("这是第二个 middleware")
	})
	//run之后会被阻塞，一般都写在这之前
	err = server.Run(":8080")
	//关闭trace也要限时，牢记谁创建的ctx谁来关，防止有人在等
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	closeFunc(ctx) //关闭trace
	//结束job定时任务
	ctx2 := expr.Stop()
	tm := time.NewTimer(time.Minute * 10)
	//防止stop信号关不掉goroutine，这里强制定时关闭
	select {
	case <-tm.C:
	case <-ctx2.Done():
	}
}

type Callbacks struct {
	SqlVector *prometheus.SummaryVec
}

func (c *Callbacks) initCallbacks(db *gorm.DB) {
	//callback会在执行sql语句前后调用，也可以用gorm的hook机制
	//作用于insert语句，在所有callback函数之前
	err := db.Callback().Create().Before("*").Register("prometheus_create_before", c.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Create().After("*").Register("prometheus_create_after", c.after("create"))
	if err != nil {
		panic(err)
	}

	err = db.Callback().Delete().Before("*").Register("prometheus_delete_before", c.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Delete().After("*").Register("prometheus_delete_after", c.after("delete"))
	if err != nil {
		panic(err)
	}

	err = db.Callback().Query().Before("*").Register("prometheus_query_before", c.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Query().After("*").Register("prometheus_query_after", c.after("query"))
	if err != nil {
		panic(err)
	}

	err = db.Callback().Raw().Before("*").Register("prometheus_raw_before", c.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Raw().After("*").Register("prometheus_raw_after", c.after("raw"))
	if err != nil {
		panic(err)
	}

	err = db.Callback().Row().Before("*").Register("prometheus_row_before", c.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Row().After("*").Register("prometheus_row_after", c.after("row"))
	if err != nil {
		panic(err)
	}
}

func (c *Callbacks) before() func(db *gorm.DB) {
	return func(db *gorm.DB) {
		//相当于ctx存储元数据，你也可以用statement的ctx
		db.Set("start_time", time.Now())
	}
}

func (c *Callbacks) after(typ string) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		data, _ := db.Get("start_time")
		startTime, ok := data.(time.Time)
		if !ok {
			return
		}
		table := db.Statement.Table
		if table == "" {
			table = "unknown"
		}
		//拿到表名
		c.SqlVector.WithLabelValues(typ, table).Observe(float64(time.Since(startTime).Milliseconds()))
	}
}

type gormLoggerFunc func(msg string, fileds ...zap.Field)

func (g gormLoggerFunc) Printf(msg string, args ...interface{}) {
	g(msg, zap.Any("gormArgs", args))
}
