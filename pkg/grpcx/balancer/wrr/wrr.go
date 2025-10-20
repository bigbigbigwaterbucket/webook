package wrr

import (
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
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
	var total int
	var maxCn *conn
	if len(p.conns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for _, cn := range p.conns {
		total += cn.weight
		cn.currentWeight += cn.weight
		if maxCn == nil || cn.currentWeight > maxCn.currentWeight {
			maxCn = cn
		}
	}
	maxCn.currentWeight -= total
	return balancer.PickResult{SubConn: maxCn.sc, Done: func(info balancer.DoneInfo) {
		// 很多动态算法，根据调用结果来调整权重，就在这里
	}}, nil
}

type conn struct {
	sc            balancer.SubConn
	weight        int
	currentWeight int
}
