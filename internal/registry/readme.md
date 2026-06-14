# 服务注册与发现

`internal/registry` 提供基于 etcd 的服务注册、发现与 gRPC resolver 能力。

## 核心设计

- 每个服务实例注册到独立的 etcd key：
  ```
  /{prefix}/{service}/{version}/{instance-id}
  ```
  例如：
  ```
  /services/Hello.Greeter/v1/127.0.0.1-50051
  /services/Hello.Greeter/v1/127.0.0.1-50052
  ```
- `instance-id` 默认使用 `Endpoint.Address` 的 sanitized 值，也可通过 `WithInstanceID("id")` 显式指定。
- `version` 为空时，默认使用 `_default`。
- 发现端通过 prefix 查询聚合所有实例，支持 `Get` 与 `Watch`。

## gRPC 下游服务启动示例

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daheige/hephfx-micro-svc/internal/registry"
	pb "github.com/daheige/hello-pb/pb/go/hello"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GreeterService struct {
	pb.UnimplementedGreeterServer
}

func (s *GreeterService) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{
		Message: fmt.Sprintf("hello %s", req.Name),
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	gs := grpc.NewServer()
	pb.RegisterGreeterServer(gs, &GreeterService{})
	reflection.Register(gs)

	reg, err := registry.NewServiceRegistry(
		[]string{"127.0.0.1:12379"},
		"services",             // etcd 前缀，最终 key 为 /services/Hello.Greeter/v1/...
		"Hello.Greeter",        // 服务名
		"v1",                   // 版本
		registry.Endpoint{
			Address:  "127.0.0.1:50051",
			Weight:   100,
			Protocol: "GRPC",
			Region:   "cn-north-1",
			Tags:     map[string]string{"version": "v1"},
			Healthy:  true,
		},
		registry.WithTTL(10),
		registry.WithEtcdTimeout(5*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := reg.Register(); err != nil {
		log.Fatal(err)
	}
	defer reg.Deregister()

	log.Println("Greeter service starting on :50051")
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down greeter service...")
}
```

## HTTP 下游服务启动示例

```go
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daheige/hephfx-micro-svc/internal/registry"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/GetUser", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"user_id": "10001",
			"name":    "张三",
			"email":   "zhangsan@example.com",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	reg, err := registry.NewServiceRegistry(
		[]string{"127.0.0.1:12379"},
		"services",
		"user",
		"v1",
		registry.Endpoint{
			Address:  "127.0.0.1:8080",
			Weight:   100,
			Protocol: "HTTP",
			Region:   "cn-north-1",
			Tags:     map[string]string{"version": "v1"},
			Healthy:  true,
		},
		registry.WithTTL(10),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := reg.Register(); err != nil {
		log.Fatal(err)
	}
	defer reg.Deregister()

	log.Println("User service starting on :8080")
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	log.Println("User service stopped")
}
```

## 服务发现

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/daheige/hephfx-micro-svc/internal/registry"
)

func main() {
	discovery, err := registry.NewEtcdDiscovery(
		[]string{"127.0.0.1:12379"},
		"services",
		5*time.Second,
	)
	if err != nil {
		log.Fatal(err)
	}

	// 一次性获取
	endpoints, err := discovery.Get(context.Background(), "Hello.Greeter", "v1")
	if err != nil {
		log.Fatal(err)
	}
	for _, ep := range endpoints {
		fmt.Println(ep.Address, ep.Weight, ep.Protocol)
	}

	// 监听变化
	watcher, err := discovery.Watch(context.Background(), "Hello.Greeter", "v1")
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Stop()

	for {
		endpoints, err := watcher.Next()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("updated endpoints:", endpoints)
	}
}
```

## gRPC Resolver

客户端可以通过自定义 scheme 直接访问 etcd 发现的服务。

```go
package main

import (
	"log"
	"time"

	"github.com/daheige/hephfx-micro-svc/internal/registry"
	pb "github.com/daheige/hello-pb/pb/go/hello"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	discovery, err := registry.NewEtcdDiscovery(
		[]string{"127.0.0.1:12379"},
		"services",
		5*time.Second,
	)
	if err != nil {
		log.Fatal(err)
	}

	// 注册 resolver，scheme 为 "etcd"
	registry.RegisterEtcdResolver(discovery, "etcd")

	conn, err := grpc.NewClient(
		"etcd:///Hello.Greeter/v1",
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewGreeterClient(conn)
	_ = client
}
```

## 配置选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `WithTTL(ttl)` | etcd 租约 TTL，单位秒 | 10 |
| `WithEtcdTimeout(timeout)` | etcd 操作超时 | 5s |
| `WithInstanceID(id)` | 自定义实例标识 | `Endpoint.Address` 的 sanitized 值 |

## 注意事项

- 包路径为 `github.com/daheige/hephfx-micro-svc/internal/registry`。
- `Endpoint.Address` 应为 `host:port` 格式，gRPC 场景不要带 `http://` 前缀。
- `Weight` 建议设置为大于 0 的值，默认语义为 100。
- `version` 为空时，注册侧与发现侧都会规范化为 `_default`，实际 etcd key 为 `/{prefix}/{service}/_default/{instance-id}`。
- 每个实例向 etcd 写入单个 `Endpoint` JSON，发现端按 `/{prefix}/{service}/{version}/` 前缀聚合所有实例。
- 发现端 `Discovery.Get` / `Watch` 会按 `Address` 去重，不校验 `Healthy` 字段，请确保注册时按需设置健康状态并在业务侧过滤。
- 多个实例只需使用不同 `Address`（或显式指定 `WithInstanceID`），即可同时注册而不互相覆盖。
- `Deregister()` 幂等安全，可重复调用；程序优雅退出时建议通过 `defer` 调用。
