# User Session Monitor

用户会话监控工具，用于监控 Linux 服务器上的用户登录和登出事件，并通过飞书机器人发送通知。

## 功能特点

- 实时监控系统认证日志
- 检测用户 SSH 登录和登出事件
- 通过飞书机器人发送即时通知
- 记录详细的事件日志
- 支持配置文件管理
- 显示服务器主机名和IP地址
- 支持多种 Linux 发行版

## 支持的系统

| 发行版           | 日志文件路径              | 备注     |
|---------------|---------------------|--------|
| Debian/Ubuntu | `/var/log/auth.log` | 默认配置   |
| CentOS/RHEL   | `/var/log/secure`   | 需要修改配置 |
| Amazon Linux  | `/var/log/secure`   | 需要修改配置 |
| SUSE          | `/var/log/messages` | 需要修改配置 |

## 安装要求

- Go 1.21 或更高版本
- Linux 系统
- 具有读取系统日志文件的权限
- 飞书机器人的 Webhook URL

## 快速开始

1. 克隆仓库：

```bash
git clone https://github.com/Annihilater/user-session-monitor.git
cd user-session-monitor
```

2. 安装依赖：

```bash
make deps
```

3. 修改配置文件：

```bash
cp config/config.yaml.example config/config.yaml
```

编辑 `config/config.yaml`，根据您的系统填入正确的日志文件路径和飞书机器人 Webhook URL：

```yaml
feishu:
  webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url"

monitor:
  # 根据您的 Linux 发行版选择正确的日志文件路径：
  # Debian/Ubuntu: /var/log/auth.log
  # CentOS/RHEL: /var/log/secure
  # Amazon Linux: /var/log/secure
  # SUSE: /var/log/messages
  log_file: "/var/log/auth.log"
```

4. 构建并运行：

```bash
# 仅构建
make build

# 构建并运行
make run
```

## 系统安装

1. 安装程序：

```bash
make install
```

这将会：

- 将程序安装到 `/usr/local/bin/`
- 创建配置目录 `/etc/user-session-monitor/`
- 复制示例配置文件

2. 安装系统服务：

```bash
make install-service
```

这将会：

- 安装 systemd 服务文件
- 提供服务管理说明

3. 启动服务：

```bash
sudo systemctl start user-session-monitor
sudo systemctl enable user-session-monitor  # 设置开机自启
```

## 通知内容

### 用户登录通知

```
用户登录通知
服务器：hostname
服务器IP：xxx.xxx.xxx.xxx
用户名：username
IP地址：xxx.xxx.xxx.xxx
登录时间：YYYY-MM-DD HH:mm:ss
```

### 用户登出通知

```
用户登出通知
服务器：hostname
服务器IP：xxx.xxx.xxx.xxx
用户名：username
登出时间：YYYY-MM-DD HH:mm:ss
```

## 开发命令

# User Session Monitor

用户会话监控工具，用于监控 Linux 服务器上的用户登录和登出事件，并通过飞书机器人发送通知。

## 功能特点

- 实时监控系统认证日志
- 检测用户 SSH 登录和登出事件
- 通过飞书机器人发送即时通知
- 记录详细的事件日志
- 支持配置文件管理
- 显示服务器主机名和IP地址
- 支持多种 Linux 发行版

## 安装要求

- Go 1.21 或更高版本
- Linux 系统
- 具有读取系统日志文件的权限
- 飞书机器人的 Webhook URL

## 快速开始

1. 克隆仓库：

```bash
git clone https://github.com/Annihilater/user-session-monitor.git
cd user-session-monitor
```

2. 修改配置文件：

```bash
cp config/config.yaml.example config/config.yaml
```

编辑 `config/config.yaml`，根据您的系统填入正确的日志文件路径和飞书机器人 Webhook URL：

```yaml
feishu:
  webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url"

monitor:
  # 根据您的 Linux 发行版选择正确的日志文件路径：
  # Debian/Ubuntu: /var/log/auth.log
  # CentOS/RHEL: /var/log/secure
  # Amazon Linux: /var/log/secure
  # SUSE: /var/log/messages
  log_file: "/var/log/auth.log"
```

3. 编译程序：

```bash
go build -o user-session-monitor cmd/monitor/main.go
```

4. 运行程序：

```bash
./user-session-monitor
```

## 通知内容

### 用户登录通知

```
用户登录通知
服务器：hostname
服务器IP：xxx.xxx.xxx.xxx
用户名：username
IP地址：xxx.xxx.xxx.xxx
登录时间：YYYY-MM-DD HH:mm:ss
```

### 用户登出通知

```
用户登出通知
服务器：hostname
服务器IP：xxx.xxx.xxx.xxx
用户名：username
登出时间：YYYY-MM-DD HH:mm:ss
```

## 使用 systemd 管理服务

1. 创建 systemd 服务文件：

```bash
sudo vim /etc/systemd/system/user-session-monitor.service
```

2. 添加以下内容：

```ini
[Unit]
Description=User Session Monitor
After=network.target

[Service]
Type=simple
User=root
ExecStart=/path/to/user-session-monitor
WorkingDirectory=/path/to/user-session-monitor
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

3. 启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable user-session-monitor
sudo systemctl start user-session-monitor
```

4. 查看服务状态：

```bash
sudo systemctl status user-session-monitor
```

## 注意事项

- 确保程序有足够的权限读取系统日志文件
- 建议使用 systemd 或其他进程管理工具来管理程序运行
- 定期检查日志确保程序正常运行
- 保护好飞书机器人的 Webhook URL，避免泄露

## 许可证

MIT License