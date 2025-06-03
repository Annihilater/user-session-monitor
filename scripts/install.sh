#!/bin/bash

# 检查是否以 root 权限运行
if [ "$EUID" -ne 0 ]; then
    echo "请使用 sudo 运行此脚本"
    exit 1
fi

# 设置变量
VERSION="v1.0.3"
ARCH=$(uname -m)
BINARY_NAME="user-session-monitor"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/user-session-monitor"
SERVICE_NAME="user-session-monitor"

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
        echo "不支持的架构: ${ARCH}"
        exit 1
        ;;
esac

# 创建临时目录
TMP_DIR=$(mktemp -d)
cd ${TMP_DIR}

echo "开始安装 ${BINARY_NAME}..."

# 下载发布包
DOWNLOAD_URL="https://github.com/Annihilater/user-session-monitor/releases/download/${VERSION}/user-session-monitor-linux-${ARCH_NAME}.tar.gz"
echo "下载发布包: ${DOWNLOAD_URL}"
if ! curl -L -o ${BINARY_NAME}.tar.gz ${DOWNLOAD_URL}; then
    echo "下载失败"
    exit 1
fi

# 解压
tar xzf ${BINARY_NAME}.tar.gz

# 安装二进制文件
install -m 755 ${BINARY_NAME} ${INSTALL_DIR}/${BINARY_NAME}

# 创建配置目录
mkdir -p ${CONFIG_DIR}

# 创建配置文件（如果不存在）
if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
    cat > ${CONFIG_DIR}/config.yaml << EOF
feishu:
  webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url"

monitor:
  # Debian/Ubuntu 使用 /var/log/auth.log
  # CentOS/RHEL/Amazon Linux 使用 /var/log/secure
  # SUSE 使用 /var/log/messages
  log_file: "/var/log/auth.log"
EOF
fi

# 创建 systemd 服务文件
cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=User Session Monitor
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
WorkingDirectory=/etc/user-session-monitor
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# 重新加载 systemd
systemctl daemon-reload

# 清理临时文件
cd /
rm -rf ${TMP_DIR}

echo "安装完成！"
echo "请编辑配置文件：${CONFIG_DIR}/config.yaml"
echo "然后运行以下命令启动服务："
echo "systemctl start ${SERVICE_NAME}"
echo "systemctl enable ${SERVICE_NAME}  # 设置开机自启" 