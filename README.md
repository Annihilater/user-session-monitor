# User Session Monitor

用户会话监控工具，用于监控 Linux 服务器上的用户登录和登出事件，并通过飞书机器人发送通知。

## 功能特点 ✨

### 实时监控 🔍

- 🚀 实时监控系统认证日志，无延迟响应
- 🔐 全面支持 SSH 登录检测（密码认证/密钥认证）
- 🌐 自动获取并显示服务器主机名和 IP 地址
- 📊 维护会话状态，智能关联登录登出事件

### 智能通知 📢

- ⚡️ 通过飞书机器人实时推送登录登出通知
- 📝 提供详细的用户、IP、时间等信息
- 🔄 自动补充登出事件缺失的会话信息
- 🎯 准确识别异常登录和非正常登出

### 系统兼容 💻

- 🐧 支持多种主流 Linux 发行版
- 📁 智能适配不同发行版的日志文件路径
- ⚙️ 灵活的配置文件管理
- 🛡️ 完善的权限和安全机制

### 可靠性保障 🛠

- 📈 详细的运行日志记录
- 🔄 服务异常自动重启
- 💾 持久化的会话状态管理
- 🔒 安全的权限控制机制

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

## 支持的登录登出事件

### 登录事件

- 支持密码认证（password）和密钥认证（publickey）
- 记录用户名、来源IP、端口等信息

### 登出事件

支持检测以下场景：

1. 用户主动断开连接
    - 执行 exit 命令
    - 执行 logout 命令
    - 按 Ctrl + D
    - SSH 客户端正常关闭

2. 网络断开或会话结束
    - SSH 会话正常结束
    - 客户端网络断开
    - 服务器端会话超时

3. 系统级别会话关闭
    - 系统强制关闭会话
    - PAM 会话超时
    - 系统关机或重启

## 开发命令

```bash
# 安装依赖
make deps

# 构建项目
make build

# 运行项目
make run

# 运行测试
make test

# 清理构建产物
make clean

# 安装到系统
make install

# 卸载
make uninstall

# 安装系统服务
make install-service
```

## 注意事项

- 确保程序有足够的权限读取系统日志文件
- 建议使用 systemd 或其他进程管理工具来管理程序运行
- 定期检查日志确保程序正常运行
- 保护好飞书机器人的 Webhook URL，避免泄露
- 程序会自动维护登录记录，用于关联登录和登出事件
- 对于某些登出场景，如果无法直接获取IP和端口信息，程序会尝试从最近的登录记录中补充信息

## 许可证

MIT License