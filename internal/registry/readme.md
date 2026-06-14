# gRPC 下游服务启动示例

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

	"github.com/daheige/hephfx-micro-svc/internal/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// OrderService 订单服务实现
type OrderService struct {
	UnimplementedOrderServiceServer
}

// CreateOrder 创建订单方法
// Bridge 会将业务方的请求路由到此方法
func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	// 业务逻辑处理
	orderID := fmt.Sprintf("order-%d", time.Now().Unix())
	return &CreateOrderResponse{
		OrderId: orderID,
		Status:  "created",
	}, nil
}

func main() {
	// 1. 创建 gRPC 服务
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	gs := grpc.NewServer()
	RegisterOrderServiceServer(gs, &OrderService{})
	reflection.Register(gs) // 启用反射，便于 Bridge 调用

	// 2. 注册到 etcd（关键步骤）
	reg, err := registry.NewServiceRegistry(
		[]string{"etcd-0:2379", "etcd-1:2379", "etcd-2:2379"},
		"/services/",           // etcd 前缀
		"order_service",        // 服务名（对应 Bridge 的 target 前缀）
		"default",              // 版本
		registry.Endpoint{
			Address:  "10.0.1.5:50051",  // 本机地址
			Weight:   100,
			Protocol: "GRPC",             // 协议类型
			Region:   "cn-north-1",
			Tags:     map[string]string{"version": "v1"},
			Healthy:  true,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := reg.Register(); err != nil {
		log.Fatal(err)
	}
	defer reg.Deregister() // 优雅退出时注销

	// 3. 启动 gRPC 服务
	log.Println("Order service starting on :50051")
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	// 4. 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down order service...")
}
```

# HTTP 下游服务启动示例

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daheige/bridge-svc/pkg/registry"
)

// UserHandler 用户服务 HTTP 处理器
type UserHandler struct{}

// GetUser 获取用户信息
// Bridge 会将 gRPC 请求转换为 HTTP 请求路由到此方法
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/user_service/"):]
	resp := map[string]interface{}{
		"user_id": userID,
		"name":    "张三",
		"email":   "zhangsan@example.com",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	// 1. 创建 HTTP 服务
	mux := http.NewServeMux()
	mux.HandleFunc("/user_service/GetUser", func(w http.ResponseWriter, r *http.Request) {
		// Bridge 会将请求路径映射为 /user_service/GetUser
		userID := r.Header.Get("x-user-id") // 从 metadata 透传的 header
		resp := map[string]interface{}{
			"user_id": userID,
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

	// 2. 注册到 etcd（关键步骤）
	reg, err := registry.NewServiceRegistry(
		[]string{"etcd-0:2379", "etcd-1:2379", "etcd-2:2379"},
		"/services/",           // etcd 前缀
		"user_service",         // 服务名（对应 Bridge 的 target）
		"default",              // 版本
		registry.Endpoint{
			Address:  "10.0.1.7:8080",   // 本机地址
			Weight:   100,
			Protocol: "HTTP",             // 协议类型（Bridge 使用 HTTP Handler）
			Region:   "cn-north-1",
			Tags:     map[string]string{"version": "v1"},
			Healthy:  true,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := reg.Register(); err != nil {
		log.Fatal(err)
	}
	defer reg.Deregister()

	// 3. 启动 HTTP 服务
	log.Println("User service starting on :8080")
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// 4. 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 5. 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	log.Println("User service stopped")
}
```

