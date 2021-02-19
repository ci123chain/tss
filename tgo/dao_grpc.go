package tgo

import (
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/resolver"
	"sync"
)

// 连接GRPC服务并自建负载均衡
type DaoGRPC struct {
	ServerName  string
	DialOptions []grpc.DialOption
}

var (
	grpcConn struct {
		m map[string]*grpc.ClientConn
		l sync.Mutex
	}
	grpcConnOne sync.Once
)

// 获取配置
func daoGRPCGetConfig(serverName string) (*ConfigPool, error) {
	poolName := "grpc-" + serverName
	config := configPoolGet(poolName)
	if config == nil {
		return nil, errors.New("pool config is null: " + poolName)
	}
	return config, nil
}

func getGrpcConn(serverName string) (conn *grpc.ClientConn, ok bool) {
	grpcConnOne.Do(func() {
		grpcConn.l.Lock()
		defer grpcConn.l.Unlock()
		if grpcConn.m == nil {
			grpcConn.m = make(map[string]*grpc.ClientConn)
		}
	})
	conn, ok = grpcConn.m[serverName]
	return
}

// 获取连接
func (p *DaoGRPC) GetConn() (conn *grpc.ClientConn, err error) {
	var ok bool
	conn, ok = getGrpcConn(p.ServerName)
	if !ok {
		grpcConn.l.Lock()
		defer grpcConn.l.Unlock()
		var config *ConfigPool
		config, err = daoGRPCGetConfig(p.ServerName)
		if err != nil {
			LogErrorw(LogNameLogic, "daoGRPCGetConfig", err)
			return
		}
		if resolver.Get(p.ServerName) == nil {
			resolver.Register(NewExampleResolverBuilder(p.ServerName, p.ServerName, config.Address))
		}
		//负载均衡
		dialOptions := append(p.DialOptions, grpc.WithBalancerName(roundrobin.Name))
		//dial
		conn, err = grpc.Dial(fmt.Sprintf("%s:///%s", p.ServerName, p.ServerName), dialOptions...)
		if err != nil {
			LogErrorw(LogNameNet, "grpc.Dial", err)
			return
		}
		//保存连接
		grpcConn.m[p.ServerName] = conn
	}
	return
}

// 关闭连接 - fake 兼容当前写法
// Deprecated: 运行中无需关闭连接
func (p *DaoGRPC) CloseConn(conn *grpc.ClientConn) (err error) {
	return
}

// 关闭连接 - shutdown时主动关闭
func (p *DaoGRPC) ShutdownConn() (err error) {
	conn, ok := getGrpcConn(p.ServerName)
	if ok {
		err = conn.Close()
	}
	return
}

type exampleResolverBuilder struct {
	exampleScheme      string
	exampleServiceName string
	addrs              []string
}

func NewExampleResolverBuilder(scheme string, serviceName string, addrs []string) (p *exampleResolverBuilder) {
	p = new(exampleResolverBuilder)
	p.exampleScheme = scheme
	p.exampleServiceName = serviceName
	p.addrs = addrs
	return
}

func (p *exampleResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &exampleResolver{
		target: target,
		cc:     cc,
		addrsStore: map[string][]string{
			p.exampleServiceName: p.addrs,
		},
	}
	r.start()
	return r, nil
}

func (p *exampleResolverBuilder) Scheme() string { return p.exampleScheme }

type exampleResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (r *exampleResolver) start() {
	addrStrs := r.addrsStore[r.target.Endpoint]
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}

func (*exampleResolver) ResolveNow(o resolver.ResolveNowOptions) {}

func (*exampleResolver) Close() {}
