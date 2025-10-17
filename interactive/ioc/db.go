package ioc

import (
	"learning_go/webook/internal/repository/dao"
	"learning_go/webook/pkg/gormx"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormPrometheus "gorm.io/plugin/prometheus"
)

func InitSrcDB() SrcDB {
	return InitDB("src")
}

func InitDstDB() DstDB {
	return InitDB("dst")
}

type SrcDB *gorm.DB
type DstDB *gorm.DB

func InitDoubleWritePool(db SrcDB, dstDB DstDB) *gormx.DoubleWritePool {
	return gormx.NewDoubleWritePool(db.ConnPool, dstDB.ConnPool, gormx.PatternSrcOnly)
}

func InitBizDB(pool *gormx.DoubleWritePool) *gorm.DB {
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: pool}))
	if err != nil {
		panic(err)
	}
	return db
}

func InitDB(key string) *gorm.DB {
	type Config struct {
		DSN string `yaml:"dsn"` //反序列化读进来的，注意大写
	}
	var config1 Config
	//远程链接etcd时，viper不支持对yaml文件分隔符.的解析！！！
	err := viper.UnmarshalKey("mysql."+key, &config1)
	if err != nil {
		panic(err)
	}
	db, err := gorm.Open(mysql.Open(config1.DSN))
	//&gorm.Config{Logger: glogger.New(gormLoggerFunc(zap.L().Debug),
	//glogger.Config{
	//	LogLevel:                  glogger.Info,
	//	SlowThreshold:             time.Millisecond * 10, //慢启动阈值，记录哪些sql执行时间慢于10ms
	//	IgnoreRecordNotFoundError: true,                  //是否忽略record没找到错误，这在某些情况下是比较常见的
	//	// ParameterizedQueries:      true,将插入的数据屏蔽掉，安全考虑
	//})},

	if err != nil {
		panic(err)
	}
	err = dao.InitTable(db)
	if err != nil {
		panic(err)
	}
	//这里提供的接口是去检测sql的一些指标
	err = db.Use(gormPrometheus.New(gormPrometheus.Config{
		DBName:          "webook_intr" + key,
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
		Subsystem: "webook_intr" + key,
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
	callbacks := Callbacks{SqlVector: sqlVector}
	callbacks.initCallbacks(db)
	return db
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
