# BENZHI_README

## 项目说明

- 项目：VanceMichael/go-label-modularbuild-g01
- 项目用途：ModularBuild coordinates prefabricated building modules from the assembly partner to the construction site. A fabricator registers a module movement, a site planner assigns it to a time-bounded lift window, quality staff release the module, and the installation crew records site-safety clearance before lifting and installation. The workflow connects planning, transport capacity, quality evidence, safety approval, installation events, audit history, and durable outbox delivery.
- Go 工具链：`golang:1.22`
- 前端工具链：无

## 标准构建、运行和测试命令

进入容器后执行：

```bash
# 编译
cd '/app' && GOTOOLCHAIN=local go build ./...

# 启动
cd '/app' && GOTOOLCHAIN=local go run ./cmd/server

# 测试
cd '/app' && GOTOOLCHAIN=local go test ./...
```

## Docker 构建和进入容器

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-task-68-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-task-68-arm64 linux/arm64
docker run -it benzhi-task-68-amd64:latest
docker run -it --platform linux/arm64 benzhi-task-68-arm64:latest
```

## 题目验证命令

1. 预期退出码 1：`go test ./internal/liftexec -run '^TestOpenSafetyHoldBlocksLiftFinalization$' -count=1`
