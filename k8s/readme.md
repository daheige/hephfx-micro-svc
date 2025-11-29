# deployment
```shell
kubectl apply -f deployment.yaml
```

# service
```shell
kubectl apply -f service.yaml
```

# port proxy
本地端口转发快速开发和调试
```shell
kubectl port-forward service/grpc-hello-svc 50051:50051 &
kubectl port-forward service/grpc-hello-svc 8090:8090 &
```

# k8s pods
查看部署的pods
```shell
kubectl get pods -o wide | grep grpc-hello-svc
```

# k8s logs
查看k8s部署的服务日志
```shell
kubectl logs -l app=grpc-hello-svc
```

# k8s resolver
运行效果如下：
![k8s-resolver.png](k8s-resolver.png)
