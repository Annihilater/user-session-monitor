# User Session Monitor

用户会话监控工具，用于监控 Linux 服务器（Debian 11）上的用户登录和登出事件，并通过飞书机器人发送通知。

## 功能特点

- 实时监控系统认证日志（`/var/log/auth.log`）
- 检测用户 SSH 登录和登出事件
- 通过飞书机器人发送即时通知
- 记录详细的事件日志
- 支持配置文件管理

## 安装要求

- Go 1.21 或更高版本
- Linux 系统（已在 Debian 11 上测试）
- 具有读取 `/var/log/auth.log` 的权限
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

编辑 `config/config.yaml`，填入您的飞书机器人 Webhook URL：

```yaml
feishu:
  webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url"

monitor:
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
用户名：username
IP地址：xxx.xxx.xxx.xxx
登录时间：YYYY-MM-DD HH:mm:ss
```

### 用户登出通知

```
用户登出通知
IP地址：xxx.xxx.xxx.xxx
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