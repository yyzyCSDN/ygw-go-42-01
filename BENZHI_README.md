# svcregistry

svcregistry 是一个分布式注册中心/服务发现控制面：服务与实例注册、心跳租约、
服务发现与负载均衡、健康检查与驱逐、版本灰度切流、注册配额与指标。

## 构建

```bash
./build_benzhi_docker.sh svcregistry linux/amd64
./build_benzhi_docker.sh svcregistry linux/arm64
```

## 运行

```bash
go run ./cmd/svcregistry
```

## 容器内验证

```bash
go build ./...
go test ./...
go vet ./...
go run ./cmd/svcregistry
```

## 功能

- 服务与实例注册/注销
- 心跳租约与存活判定
- 服务发现与负载均衡（轮询/一致性哈希/最小连接）
- 健康检查（主动探测 + 被动失败计数）
- 过期租约与不健康实例驱逐
- 版本灰度与权重切流
- 注册配额限流与指标
