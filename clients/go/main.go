package main

import (
	"context"
	"log"

	"github.com/daheige/hello-pb/pb"
	"github.com/daheige/hephfx/micro/gclient"
)

func main() {
	address := "127.0.0.1:50051"
	// 或者使用k8s命名服务地址，例如:hello.test.svc.cluster.local:50051
	// 使用k8s命名服务+dns解析方式连接，格式:dns:///your-service.namespace.svc.cluster.local:50051
	// address := "dns:///hello.default.svc.cluster.local:30051"
	// address := "hello.default.svc.cluster.local:30051"
	log.Println("address: ", address)

	client, err := gclient.InitGRPCClient(address, pb.NewGreeterClient)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer gclient.Close(address)

	// Contact the server and print out its response.
	for i := 0; i < 3; i++ {
		res, err := client.SayHello(context.Background(), &pb.HelloReq{
			Name: "daheige",
		})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}

		log.Printf("res message:%s", res.Message)
	}
}
