.PHONY: build clean run install test

# 二进制文件名
BINARY=user-session-monitor

# 主程序入口
MAIN_GO=cmd/monitor/main.go

# Go 相关命令
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# 编译参数
LDFLAGS=-ldflags "-s -w"

# 默认目标
all: build

# 构建项目
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY) $(MAIN_GO)

# 构建并运行
run: build
	./$(BINARY)

# 清理构建产物
clean:
	$(GOCLEAN)
	rm -f $(BINARY)

# 安装依赖
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# 运行测试
test:
	$(GOTEST) -v ./...

# 安装到系统
install: build
	sudo cp $(BINARY) /usr/local/bin/
	sudo mkdir -p /etc/user-session-monitor
	sudo cp -n config/config.yaml.example /etc/user-session-monitor/config.yaml
	@echo "安装完成！请修改配置文件: /etc/user-session-monitor/config.yaml"

# 卸载
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY)
	sudo rm -rf /etc/user-session-monitor

# 创建 systemd 服务
install-service: install
	sudo cp scripts/user-session-monitor.service /etc/systemd/system/
	sudo systemctl daemon-reload
	@echo "服务安装完成！使用以下命令管理服务："
	@echo "启动服务: sudo systemctl start user-session-monitor"
	@echo "停止服务: sudo systemctl stop user-session-monitor"
	@echo "查看状态: sudo systemctl status user-session-monitor"
	@echo "开机自启: sudo systemctl enable user-session-monitor" 