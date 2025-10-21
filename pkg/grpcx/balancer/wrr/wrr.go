package wrr

import (
	"context"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const name = "custom_wrr"

func init() {
	balancer.Register(base.NewBalancerBuilder(name, &PickerBuilder{}, base.Config{HealthCheck: true})) //健康检查连接，服务端崩了就去掉
}

type PickerBuilder struct {
}

func (p *PickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	conns := make([]*conn, 0, len(info.ReadySCs))
	for sc, scInfo := range info.ReadySCs {
		cn := &conn{
			sc: sc,
		}
		mt, ok := scInfo.Address.Metadata.(map[string]any)
		if ok {
			weightVal := mt["weight"]
			weight, _ := weightVal.(float64)
			cn.weight = int(weight)
			cn.currentWeight = cn.weight //初始化值为weight
		}
		conns = append(conns, cn)
	}
	return &Picker{conns: conns}
}

type Picker struct {
	conns []*conn
	mutex sync.Mutex
}

func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	//group := info.Ctx.Value("group") // 可以这样从 context 取用户请求分组
	//info带的context是grpc的context
	var total int
	var maxCn *conn
	if len(p.conns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for _, cn := range p.conns {
		if !cn.available {
			continue
		}
		total += cn.weight
		cn.currentWeight += cn.weight
		if maxCn == nil || cn.currentWeight > maxCn.currentWeight {
			maxCn = cn
		}
	}
	maxCn.currentWeight -= total
	return balancer.PickResult{SubConn: maxCn.sc, Done: func(info balancer.DoneInfo) {
		// 很多动态算法，根据调用结果来调整权重，就在这里
		err := info.Err
		if err == nil {
			//可以考虑提高权重
			return
		} else {
			switch err {
			case context.Canceled:
				return
			case context.DeadlineExceeded:
				//可以考虑降低权重
				return
			case io.EOF, io.ErrUnexpectedEOF:
				//服务端调用 grpc.Server.Stop() 或 grpc.Server.GracefulStop()
				//或者由于服务端进程退出、容器重启。
				//总之代表“连接”已经断开了
				maxCn.available = false
			default:
				st, ok := status.FromError(err)
				if ok {
					code := st.Code()
					switch code {
					case codes.Unavailable:
						//可能是熔断
						maxCn.available = false
						go func() {
							for i := 0; i < 10; i++ {
								time.Sleep(1000)
								if p.HealthCheck(maxCn) {
									maxCn.available = true
								}
							}
						}()
					case codes.ResourceExhausted:
						//可能是限流
						// 最好是 currentWeight 和 weight 都调低
						// 减少它被选中的概率
					}
				}
			}
		}
	}}, nil
}

func (p *Picker) HealthCheck(cn *conn) bool {
	return true
}

type conn struct {
	sc            balancer.SubConn
	weight        int
	currentWeight int
	available     bool
	group         string //为分组考虑，加入有vip客户端节点
}
