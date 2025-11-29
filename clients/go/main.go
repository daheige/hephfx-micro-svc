package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/daheige/hephfx-micro-svc/pb"
)

func main() {
	address := "localhost:50051"
	// 或者使用k8s命名服务地址，例如:hello.test.svc.cluster.local:50051
	// 使用k8s命名服务+dns解析方式连接，格式:dns:///your-service.namespace.svc.cluster.local:50051
	// address := "dns:///hello.test.svc.cluster.local:50051"
	// address := "hello.test.svc.cluster.local:50051"
	log.Println("address: ", address)

	// Set up a connection to the server.
	clientConn, err := grpc.NewClient(
		address,
		// 如果使用k8s命名服务以及headless方式访问，需要打开下面的注释，实现客户端负载均衡
		// grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
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
