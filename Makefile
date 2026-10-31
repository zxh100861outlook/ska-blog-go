# ska-blog-go 开发/构建入口（制表符缩进，勿用空格）
BIN := bin/sshblog
GO  := go

.PHONY: build test vet run tidy deploy-arm64 clean

# 编译主程序到 bin/sshblog
build:
	mkdir -p $(dir $(BIN))
	$(GO) build -o $(BIN) .

# 运行全部测试
test:
	$(GO) test ./...

# 静态检查
vet:
	$(GO) vet ./...

# 本地直接跑（SSH 服务器，默认 127.0.0.1:2222）
run:
	$(GO) run .

# 整理 go.mod/go.sum
tidy:
	$(GO) mod tidy

# 交叉编译到 linux/arm64，产出二进制并 tar 打包（上传到板子用）
deploy-arm64:
	mkdir -p /tmp/sshblog-deploy
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o /tmp/sshblog-deploy/sshblog .
	tar -czf /tmp/sshblog-deploy/sshblog-arm64.tar.gz -C /tmp/sshblog-deploy sshblog

# 清理构建产物
clean:
	rm -rf bin
