#!/bin/bash

# 输出带时间戳的日志
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# 输出带时间戳的错误日志
error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] 错误: $1" >&2
}

# 检查是否以 root 权限运行
if [ "$EUID" -ne 0 ]; then
    error "请使用 sudo 运行此脚本"
    exit 1
fi

# 设置变量
VERSION="v1.0.9"
ARCH=$(uname -m)
BINARY_NAME="user-session-monitor"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/user-session-monitor"
SERVICE_NAME="user-session-monitor"

log "开始安装 ${BINARY_NAME} ${VERSION}"
log "系统架构: ${ARCH}"

# 根据架构选择下载文件
case ${ARCH} in
    x86_64)
        ARCH_NAME="amd64"
        ;;
    aarch64)
        ARCH_NAME="arm64"
        ;;
    armv7l)
        ARCH_NAME="armv7"
        ;;
    armv6l)
        ARCH_NAME="armv6"
        ;;
    i386|i686)
        ARCH_NAME="386"
        ;;
    *)
        error "不支持的架构: ${ARCH}"
        exit 1
        ;;
esac

log "选择的架构包: linux-${ARCH_NAME}"

# 创建临时目录
TMP_DIR=$(mktemp -d)
log "创建临时目录: ${TMP_DIR}"
cd ${TMP_DIR}

# 下载发布包
DOWNLOAD_URL="https://github.com/Annihilater/user-session-monitor/releases/download/${VERSION}/user-session-monitor-linux-${ARCH_NAME}.tar.gz"
log "开始下载发布包..."
log "下载地址: ${DOWNLOAD_URL}"
if ! curl -L -o ${BINARY_NAME}.tar.gz ${DOWNLOAD_URL}; then
    error "下载失败，请检查网络连接或版本号是否正确"
    exit 1
fi
log "下载完成"

# 解压
log "解压发布包..."
if ! tar xzf ${BINARY_NAME}.tar.gz; then
    error "解压失败"
    exit 1
fi
log "解压完成"

# 安装二进制文件
log "安装二进制文件到 ${INSTALL_DIR}/${BINARY_NAME}"
if ! install -m 755 ${BINARY_NAME} ${INSTALL_DIR}/${BINARY_NAME}; then
    error "安装二进制文件失败"
    exit 1
fi
log "二进制文件安装完成"

# 创建配置目录
log "创建配置目录: ${CONFIG_DIR}"
if ! mkdir -p ${CONFIG_DIR}; then
    error "创建配置目录失败"
    exit 1
fi

# 创建配置文件（如果不存在）
if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
    log "创建默认配置文件: ${CONFIG_DIR}/config.yaml"
    cat > ${CONFIG_DIR}/config.yaml << EOF
feishu:
  webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url"

monitor:
  # Debian/Ubuntu 使用 /var/log/auth.log
  # CentOS/RHEL/Amazon Linux 使用 /var/log/secure
  # SUSE 使用 /var/log/messages
  log_file: "/var/log/auth.log"
EOF
    if [ $? -ne 0 ]; then
        error "创建配置文件失败"
        exit 1
    fi
    log "配置文件创建完成"
else
    log "配置文件已存在，跳过创建"
fi

# 创建 systemd 服务文件
log "创建 systemd 服务文件: /etc/systemd/system/${SERVICE_NAME}.service"
cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=User Session Monitor
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${BINARY_NAME} -config ${CONFIG_DIR}/config.yaml
WorkingDirectory=/etc/user-session-monitor
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
if [ $? -ne 0 ]; then
    error "创建服务文件失败"
    exit 1
fi
log "服务文件创建完成"

# 重新加载 systemd
log "重新加载 systemd 配置..."
if ! systemctl daemon-reload; then
    error "重新加载 systemd 配置失败"
    exit 1
fi
log "systemd 配置重新加载完成"

# 清理临时文件
log "清理临时文件..."
cd /
rm -rf ${TMP_DIR}
log "清理完成"

log "安装成功完成！"
echo
log "后续步骤："
echo "1. 编辑配置文件：${CONFIG_DIR}/config.yaml"
echo "   - 设置飞书机器人的 webhook URL"
echo "   - 根据系统类型设置正确的日志文件路径"
echo
echo "2. 启动服务："
echo "   systemctl start ${SERVICE_NAME}"
echo "   systemctl enable ${SERVICE_NAME}  # 设置开机自启"
echo
echo "3. 查看服务状态："
echo "   systemctl status ${SERVICE_NAME}"
echo
log "如需帮助，请访问: https://github.com/Annihilater/user-session-monitor" 