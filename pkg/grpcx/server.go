package grpcx

import (
	"context"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"learning_go/webook/pkg/netx"
	"net"
	"strconv"
	"time"
)

type Server struct {
	*grpc.Server
	Name            string
	EtcdAddrs       []string
	Port            int
	client          *etcdv3.Client
	kaCancel        func()
	endPointManager endpoints.Manager
	key             string
}

//func NewServer(client  *etcdv3.Client) *Server {
//	return &Server{}
//}

func (s *Server) Serve() error {
	l, err := net.Listen("tcp", ":"+strconv.Itoa(s.Port))
	if err != nil {
		panic(err)
	}
	err = s.Register()
	if err != nil {
		panic(err)
	}
	return s.Server.Serve(l)
}

func (s *Server) Register() error {
	client, err := etcdv3.New(etcdv3.Config{
		Endpoints: s.EtcdAddrs,
	})
	if err != nil {
		return err
	}
	s.client = client
	em, err := endpoints.NewManager(s.client, "service/"+s.Name)
	if err != nil {
		zap.L().Error("", zap.Error(err))
		return err
	}
	s.endPointManager = em
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	//注册在etcd里的服务的ip必须能够被外部访问
	addr := netx.GetOutIp() + ":" + strconv.Itoa(s.Port)
	key := "service/" + s.Name + "/" + addr
	s.key = key
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	//ttl 单位是 s
	//每隔 1/3 ttl 就续约
	var ttl int64 = 15
	leaseResp, err := s.client.Grant(ctx, ttl)
	if err != nil {
		zap.L().Error("", zap.Error(err))
		return err
	}
	//try to connect to the etcd client and register the endpoint, e.g. RPC service.
	err = em.AddEndpoint(ctx, key, endpoints.Endpoint{Addr: addr}, etcdv3.WithLease(leaseResp.ID)) //!! upsert语义，服务key已经存在就会更新
	if err != nil {
		zap.L().Error("", zap.Error(err))
		return err
	}
	kaCtx, kaCancel := context.WithCancel(context.Background())
	s.kaCancel = kaCancel
	go func() {
		ch, er := s.client.KeepAlive(kaCtx, leaseResp.ID) //自动开启goroutine去续约，续约时间取决于ttl
		if er != nil {
			zap.L().Error("", zap.Error(er))
		}
		for kaResp := range ch {
			zap.L().Info(kaResp.String() + time.Now().String())
		}
	}()

	//update metadata
	go func() {
		ticker := time.NewTicker(time.Second)
		for now := range ticker.C {
			er := em.AddEndpoint(context.Background(), key, endpoints.Endpoint{Addr: addr, Metadata: now.String()},
				etcdv3.WithLease(leaseResp.ID))
			if er != nil {
				zap.L().Error("", zap.Error(er))
			}
		}
	}()
	return nil
}

func (s *Server) Cancel() {
	s.kaCancel()
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	err := s.endPointManager.DeleteEndpoint(ctx2, s.key)
	if err != nil {
		zap.L().Error("", zap.Error(err))
	}
	cancel2()
	s.client.Close()
}
