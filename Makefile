# 声明伪目标
.PHONY: all build clean run test check install uninstall prod dev prod-run dev-run prod-check dev-check prod-start dev-start prod-stop dev-stop prod-restart dev-restart prod-log dev-log status

# 项目信息
PROJECT_NAME := user-session-monitor
VERSION      := 1.0.0

# 目录结构
CMD_DIR      := cmd
CONFIG_DIR   := config
BUILD_DIR    := build
SCRIPTS_DIR  := scripts

# 构建目标
BINARY       := $(PROJECT_NAME)
MAIN_GO      := $(CMD_DIR)/monitor/main.go

# Go 工具链
GO          := go
GO_BUILD    := $(GO) build
GO_CLEAN    := $(GO) clean
GO_TEST     := $(GO) test
GO_GET      := $(GO) get
GO_MOD      := $(GO) mod

# 编译参数
LDFLAGS     := -ldflags "-s -w -X main.Version=$(VERSION)"

# 安装路径
INSTALL_DIR      := /usr/local/bin
CONFIG_INST_DIR  := /etc/$(PROJECT_NAME)

# 进程信息
PROCESS_NAME    := $(PROJECT_NAME)

# 默认目标
all: build

# 构建目标
build: $(BUILD_DIR)
	@echo "==> 构建项目 $(BINARY)..."
	@$(GO_BUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(MAIN_GO)
	@ls -l $(BUILD_DIR)/$(BINARY)
	@echo "==> 构建完成！"

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# =====================
# 环境别名（默认使用生产环境）
# =====================
run: prod-run
check: prod-check
start: prod-start
stop: prod-stop
restart: prod-restart
log: prod-log

# =====================
# 生产环境命令
# =====================

# 生产环境运行
prod-run: build
	@echo "==> [生产环境] 运行 $(BINARY)..."
	@$(BUILD_DIR)/$(BINARY) run

# 生产环境检查状态
prod-check: build
	@echo "==> [生产环境] 检查服务状态..."
	@$(BUILD_DIR)/$(BINARY) check

# 生产环境启动服务
prod-start: build
	@echo "==> [生产环境] 启动服务..."
	@$(BUILD_DIR)/$(BINARY) start

# 生产环境停止服务
prod-stop: clean-process
	@echo "==> [生产环境] 停止服务..."
	@$(BUILD_DIR)/$(BINARY) stop

# 生产环境重启服务
prod-restart: prod-stop prod-start
	@echo "==> [生产环境] 重启完成"

# 生产环境查看日志
prod-log:
	@echo "==> [生产环境] 查看服务日志..."
	@$(BUILD_DIR)/$(BINARY) log

# =====================
# 开发环境命令
# =====================

# 开发环境运行
dev-run: clean-process
	@echo "==> [开发环境] 运行服务..."
	@$(GO) run $(MAIN_GO) run

# 开发环境检查状态
dev-check:
	@echo "==> [开发环境] 检查服务状态..."
	@$(GO) run $(MAIN_GO) check

# 开发环境启动服务
dev-start: clean-process
	@echo "==> [开发环境] 启动服务..."
	@$(GO) run $(MAIN_GO) start

# 开发环境停止服务
dev-stop: clean-process
	@echo "==> [开发环境] 停止服务..."
	@$(GO) run $(MAIN_GO) stop

# 开发环境重启服务
dev-restart: dev-stop dev-start
	@echo "==> [开发环境] 重启完成"

# 开发环境查看日志
dev-log:
	@echo "==> [开发环境] 查看服务日志..."
	@$(GO) run $(MAIN_GO) log

# =====================
# 进程状态检查
# =====================
status:
	@echo "==> 检查 $(PROCESS_NAME) 进程状态..."
	@ps aux | grep -v grep | grep $(PROCESS_NAME) || echo "进程未运行"

# =====================
# 维护命令
# =====================

# 清理目标
clean:
	@echo "==> 清理构建产物..."
	@$(GO_CLEAN)
	@rm -rf $(BUILD_DIR)

# 依赖管理
deps:
	@echo "==> 安装依赖..."
	@$(GO_MOD) download
	@$(GO_MOD) tidy

# 测试目标
test:
	@echo "==> 运行测试..."
	@$(GO_TEST) -v ./...

# =====================
# 安装相关命令
# =====================

# 安装目标
install: build
	@echo "==> 安装 $(BINARY) 到系统..."
	@sudo install -d $(CONFIG_INST_DIR)
	@sudo install -m 755 $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)
	@if [ ! -f $(CONFIG_INST_DIR)/config.yaml ]; then \
		sudo cp $(CONFIG_DIR)/config.yaml $(CONFIG_INST_DIR)/config.yaml; \
	fi
	@echo "==> 安装完成！请修改配置文件: $(CONFIG_INST_DIR)/config.yaml"

# 卸载目标
uninstall:
	@echo "==> 卸载 $(BINARY)..."
	@sudo rm -f $(INSTALL_DIR)/$(BINARY)
	@sudo rm -rf $(CONFIG_INST_DIR)

# 安装系统服务
install-service: install
	@echo "==> 安装 systemd 服务..."
	@sudo cp $(SCRIPTS_DIR)/user-session-monitor.service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@echo "==> 服务安装完成！使用以下命令管理服务："
	@echo "    启动服务: sudo systemctl start user-session-monitor"
	@echo "    停止服务: sudo systemctl stop user-session-monitor"
	@echo "    查看状态: sudo systemctl status user-session-monitor"
	@echo "    开机自启: sudo systemctl enable user-session-monitor"

# =====================
# 内部工具命令
# =====================

# 清理进程
clean-process:
	@echo "==> 清理进程..."
	@if [ -f /var/run/$(PROJECT_NAME).pid ]; then \
		pid=$$(cat /var/run/$(PROJECT_NAME).pid); \
		if ps -p $$pid >/dev/null 2>&1; then \
			echo "==> 停止进程 (PID: $$pid)"; \
			kill $$pid || true; \
			sleep 1; \
		fi; \
		rm -f /var/run/$(PROJECT_NAME).pid; \
	fi 