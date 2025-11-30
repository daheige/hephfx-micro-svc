package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"

	"github.com/daheige/hephfx-micro-svc/pb"
)

// 自定义k8s解析器，用于Kubernetes Headless Service
type k8sResolver struct {
	target resolver.Target
	cc     resolver.ClientConn
}

func (r *k8sResolver) ResolveNow(resolver.ResolveNowOptions) {
	// 实现DNS解析逻辑
	addrs := []resolver.Address{
		{Addr: "hello.default.svc.cluster.local:50051"},
	}

	r.cc.UpdateState(resolver.State{Addresses: addrs})
}

func (r *k8sResolver) Close() {}

type k8sResolverBuilder struct{}

// Build 实现build方法
func (b *k8sResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &k8sResolver{
		target: target,
		cc:     cc,
	}

	r.ResolveNow(resolver.ResolveNowOptions{})
	return r, nil
}

func (b *k8sResolverBuilder) Scheme() string {
	return "k8s"
}

// 注册自定义解析器
func init() {
	resolver.Register(&k8sResolverBuilder{})
}

func main() {
	// address := "localhost:50051"
	// 使用k8s命名服务访问，这里是使用自定义的k8s dns解析模式
	address := "dns:///hello.default.svc.cluster.local:50051"
	log.Println("address: ", address)

	// Set up a connection to the server.
	clientConn, err := grpc.NewClient(
		address,
		// 如果使用k8s命名服务以及headless方式访问
		// 关键配置：启用round_robin负载均衡策略
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	defer func() {
		_ = clientConn.Close()
	}()

	client := pb.NewGreeterClient(clientConn)

	// Contact the server and print out its response.
	for i := 0; i < 100; i++ {
		res, err := client.SayHello(context.Background(), &pb.HelloReq{
			Name: "daheige",
		})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}

		log.Printf("res message:%s", res.Message)
	}
}
