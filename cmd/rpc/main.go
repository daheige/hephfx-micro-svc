package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/daheige/hephfx/logger"
	"github.com/daheige/hephfx/micro"
	"github.com/daheige/hephfx/monitor"

	"github.com/daheige/hello-pb/pb"

	"github.com/daheige/registry"
	"github.com/daheige/registry/etcd"

	"github.com/daheige/hephfx-micro-svc/internal/interfaces/rpc/interceptor"
)

func main() {
	// 初始化日志配置
	logger.Default(
		logger.WithStdout(true),
		logger.WithJsonFormat(true),
		logger.WithWriteToFile(false),
	)

	grpcPort := 50051
	// 如果启动了grpc http gateway，可以自定义路由地址，例如用于健康检查
	// 访问地址：http://localhost:50051/healthz
	// routerOpts := micro.WithRoutes(micro.Route{
	// 	Method: "GET",
	// 	Path:   "/healthz",
	// 	Handler: func(w http.ResponseWriter, r *http.Request) {
	// 		log.Println("request path:", r.URL.Path)
	// 		b, _ := json.Marshal(map[string]interface{}{
	// 			"code":    0,
	// 			"message": "OK",
	// 			"active":  true,
	// 		})
	// 		w.Write(b)
	// 	},
	// })

	// 创建grpc微服务实例
	address := fmt.Sprintf(":%d", grpcPort)
	s := micro.NewService(
		address,

		// start grpc and http gateway use one address
		micro.WithEnableGRPCShareAddress(),
		// micro.WithGRPCHTTPAddress(fmt.Sprintf("0.0.0.0:%d", 8080)),
		micro.WithHandlerFromEndpoints(pb.RegisterGreeterHandlerFromEndpoint), // register http endpoint
		// routerOpts, // register custom router

		micro.WithLogger(micro.LoggerFunc(log.Printf)),
		micro.WithShutdownTimeout(5*time.Second),
		micro.WithEnablePrometheus(), // prometheus interceptor

		micro.WithEnableRequestValidator(), // request validator interceptor
		// 使用自定义请求拦截器
		micro.WithUnaryInterceptor(interceptor.AccessLog),
		micro.WithShutdownFunc(func() {
			time.Sleep(3 * time.Second) // mock long operations
			log.Println("grpc server shutdown")
		}),
	)

	// 初始化prometheus和pprof，可以根据实际情况更改
	// metrics访问地址：http://localhost:8090/metrics
	// pprof访问地址：http://localhost:8090/debug/pprof
	// 在metrics接口可以搜索grpc_开头的指标，表示当前微服务接口运行情况
	monitor.InitMonitor(8090)

	// 初始化greeter service
	service := &GreeterServer{}

	// 注册grpc微服务
	pb.RegisterGreeterServer(s.GRPCServer, service)

	regAddr, err := registry.Resolve(address)
	if err != nil {
		log.Fatal("resolve address err:", err)
	}

	// etcd注册
	regEntry, err := etcd.NewServiceRegistry([]string{
		"127.0.0.1:12379",
	}, "services", "Hello.Greeter", registry.Endpoint{
		Address:  regAddr,
		Weight:   100,
		Protocol: registry.ProtocolGRPC,
		Region:   "",
		Version:  "", // 对应的版本号，例如: v1,v2,或者为空
		Healthy:  true,
	}, etcd.WithTTL(10), etcd.WithEtcdTimeout(5*time.Second))
	if err != nil {
		log.Fatal("failed to new service registry", err)
	}
	err = regEntry.Register()
	if err != nil {
		log.Fatal("failed to register service", err)
	}
	defer regEntry.Deregister()

	// 运行服务
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}

// GreeterServer 实现greeter微服务
type GreeterServer struct {
	// 这里必须包含这个解构体才可以，否则就是没有实现
	pb.UnimplementedGreeterServer
}

// SayHello 实现say hello方法
func (s *GreeterServer) SayHello(ctx context.Context, req *pb.HelloReq) (*pb.HelloReply, error) {
	reply := &pb.HelloReply{
		Message: fmt.Sprintf("hello,%s", req.Name),
	}

	return reply, nil
}

// Healthz 实现健康检查
func (s *GreeterServer) Healthz(context.Context, *pb.HealthzReq) (*pb.HealthzReply, error) {
	reply := &pb.HealthzReply{
		CurrentTime: time.Now().Format("2006-01-02 15:04:05"),
		Alive:       true,
	}

	return reply, nil
}
