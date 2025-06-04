.PHONY: build clean run install test check start stop restart log run-debug check-debug start-debug restart-debug clean-process

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

build:
	rm -f $(BINARY)
	@echo "构建项目 $(BINARY)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY) $(MAIN_GO)
	ls -l $(BINARY)
	@echo "构建完成！"

run: build
	@echo "构建并运行 $(BINARY)..."
	./$(BINARY)

run-debug: clean-process
	@echo "直接运行源码（用于调试） $(MAIN_GO)..."
	go run $(MAIN_GO)

check: build
	@echo "检查服务状态..."
	./$(BINARY) check

check-debug:
	@echo "直接运行源码检查状态（用于调试）..."
	go run $(MAIN_GO) check

start: build
	@echo "启动服务..."
	./$(BINARY) start

start-debug: clean-process
	@echo "直接运行源码启动服务（用于调试）..."
	go run $(MAIN_GO) start

stop: clean-process
	@echo "停止服务..."
	./$(BINARY) stop

restart: build
	@echo "重启服务..."
	./$(BINARY) restart

restart-debug: clean-process
	@echo "直接运行源码重启服务（用于调试）"
	go run $(MAIN_GO) restart

log:
	@echo "查看服务日志..."
	./$(BINARY) log

clean:
	@echo "清理构建产物..."
	$(GOCLEAN)
	rm -f $(BINARY)

deps:
	@echo "安装依赖..."
	$(GOMOD) download
	$(GOMOD) tidy

test:
	@echo "运行测试..."
	$(GOTEST) -v ./...

install: build
	@echo "安装 $(BINARY) 到系统..."
	sudo cp $(BINARY) /usr/local/bin/
	sudo mkdir -p /etc/user-session-monitor
	sudo cp -n config/config.yaml /etc/user-session-monitor/config.yaml
	@echo "安装完成！请修改配置文件: /etc/user-session-monitor/config.yaml"

uninstall:
	@echo "卸载 $(BINARY)..."
	sudo rm -f /usr/local/bin/$(BINARY)
	sudo rm -rf /etc/user-session-monitor

install-service: install
	@echo "安装 systemd 服务..."
	sudo cp scripts/user-session-monitor.service /etc/systemd/system/
	sudo systemctl daemon-reload
	@echo "服务安装完成！使用以下命令管理服务："
	@echo "启动服务: sudo systemctl start user-session-monitor"
	@echo "停止服务: sudo systemctl stop user-session-monitor"
	@echo "查看状态: sudo systemctl status user-session-monitor"
	@echo "开机自启: sudo systemctl enable user-session-monitor"

# 清理进程
clean-process:
	@if [ -f /var/run/user-session-monitor.pid ]; then \
		pid=$$(cat /var/run/user-session-monitor.pid); \
		if ps -p $$pid > /dev/null; then \
			echo "Stopping existing process (PID: $$pid)..."; \
			kill $$pid; \
			sleep 1; \
		fi; \
		rm -f /var/run/user-session-monitor.pid; \
	fi 