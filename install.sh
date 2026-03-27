#!/bin/bash
#
# node-push-exporter 安装脚本
# 功能：安装 node_exporter 和 node-push-exporter
# 使用: curl -sL https://example.com/install.sh | sudo bash -s -- --pushgateway http://pushgateway:9091
#

set -e

# ============ 默认配置 ============
PUSHGATEWAY_URL="http://localhost:9091"
PUSH_INTERVAL=60
JOB_NAME="node"
INSTANCE_NAME=""
NODE_EXPORTER_VERSION="1.8.1"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/node-push-exporter"
BINARY_NAME="node-push-exporter"
EXPORTER_NAME="node_exporter"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

# ============ 显示帮助 ============
show_help() {
    cat << EOF
node-push-exporter 安装脚本

用法: curl -sL <install-url> | sudo bash -s -- [选项]

选项:
  --pushgateway URL    Pushgateway地址 (默认: http://localhost:9091)
  --interval SECS      推送间隔秒数 (默认: 60)
  --job NAME           任务名称 (默认: node)
  --instance NAME      实例名称 (默认: 主机名)
  --exporter-version VERSION  node_exporter版本 (默认: 1.8.1)
  --help               显示帮助信息

示例:
  # 默认安装
  curl -sL <install-url> | sudo bash

  # 指定 Pushgateway
  curl -sL <install-url> | sudo bash -s -- --pushgateway http://prometheus:9091

  # 完整参数
  curl -sL <install-url> | sudo bash -s -- \\
    --pushgateway http://prometheus:9091 \\
    --interval 30 \\
    --job mynode
EOF
}

# ============ 解析参数 ============
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --pushgateway) PUSHGATEWAY_URL="$2"; shift 2 ;;
            --interval) PUSH_INTERVAL="$2"; shift 2 ;;
            --job) JOB_NAME="$2"; shift 2 ;;
            --instance) INSTANCE_NAME="$2"; shift 2 ;;
            --exporter-version) NODE_EXPORTER_VERSION="$2"; shift 2 ;;
            --help) show_help; exit 0 ;;
            *) log_error "未知选项: $1"; show_help; exit 1 ;;
        esac
    done
}

# ============ 检测系统 ============
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
    elif [[ -f /etc/redhat-release ]]; then
        OS="rhel"
    elif [[ -f /etc/debian_version ]]; then
        OS="debian"
    else
        OS="linux"
    fi

    # 检测架构
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH_NAME="amd64" ;;
        aarch64|arm64) ARCH_NAME="arm64" ;;
        armv7l) ARCH_NAME="armv7" ;;
        *) log_error "不支持的架构: $ARCH"; exit 1 ;;
    esac

    # 检查 systemd
    if command -v systemctl &> /dev/null && systemctl --version &> /dev/null; then
        SYSTEMD=true
    else
        SYSTEMD=false
    fi

    log_info "操作系统: $OS, 架构: $ARCH_NAME, systemd: $SYSTEMD"
}

# ============ 安装 node_exporter ============
install_node_exporter() {
    log_step "安装 node_exporter ${NODE_EXPORTER_VERSION}..."

    # 检查是否已安装
    if command -v $EXPORTER_NAME &> /dev/null; then
        log_warn "node_exporter 已安装，跳过安装"
        return 0
    fi

    # 下载 URL
    DOWNLOAD_URL="https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/node_exporter-${NODE_EXPORTER_VERSION}.linux-${ARCH_NAME}.tar.gz"

    # 创建临时目录
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"

    log_info "下载 node_exporter..."
    if ! curl -sL "$DOWNLOAD_URL" -o node_exporter.tar.gz; then
        log_error "下载失败，请检查版本号是否正确: $NODE_EXPORTER_VERSION"
        exit 1
    fi

    log_info "解压并安装..."
    tar xzf node_exporter.tar.gz
    cp node_exporter-${NODE_EXPORTER_VERSION}.linux-${ARCH_NAME}/$EXPORTER_NAME $INSTALL_DIR/
    chmod +x $INSTALL_DIR/$EXPORTER_NAME

    # 清理
    cd /
    rm -rf "$TEMP_DIR"

    log_info "node_exporter 安装完成"
}

# ============ 安装 node-push-exporter ============
install_exporter() {
    log_step "安装 node-push-exporter..."

    # 确定二进制文件路径
    LOCAL_BINARY="./${BINARY_NAME}"

    if [[ -f "$LOCAL_BINARY" ]]; then
        log_info "使用本地构建的二进制文件"
        cp "$LOCAL_BINARY" $INSTALL_DIR/
    else
        log_error "未找到二进制文件: $LOCAL_BINARY"
        log_info "请先构建项目: go build -o node-push-exporter ./src/"
        exit 1
    fi

    chmod +x $INSTALL_DIR/$BINARY_NAME
    log_info "node-push-exporter 安装完成"
}

# ============ 创建配置目录 ============
create_directories() {
    log_step "创建目录..."
    mkdir -p $CONFIG_DIR
}

# ============ 创建配置文件 ============
create_config() {
    log_step "创建配置文件..."

    # 生成默认实例名
    if [[ -z "$INSTANCE_NAME" ]]; then
        INSTANCE_NAME=$(hostname)
    fi

    cat > $CONFIG_DIR/config.yaml << EOF
# node-push-exporter 配置

# Pushgateway 设置
pushgateway.url=${PUSHGATEWAY_URL}
pushgateway.job=${JOB_NAME}
pushgateway.instance=${INSTANCE_NAME}
pushgateway.interval=${PUSH_INTERVAL}
pushgateway.timeout=10

# node_exporter 设置
node_exporter.path=${INSTALL_DIR}/${EXPORTER_NAME}
node_exporter.port=9100
node_exporter.metrics_url=http://localhost:9100/metrics
EOF

    log_info "配置文件: $CONFIG_DIR/config.yaml"
}

# ============ 创建 systemd 服务 ============
create_systemd_service() {
    if [[ "$SYSTEMD" == "false" ]]; then
        log_warn "systemd 不可用，跳过服务配置"
        return
    fi

    log_step "创建 systemd 服务..."

    cat > /etc/systemd/system/node-push-exporter.service << EOF
[Unit]
Description=Node Metrics Exporter (via node_exporter)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=${INSTALL_DIR}/${BINARY_NAME} --config ${CONFIG_DIR}/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=node-push-exporter

# 安全加固
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    log_info "systemd 服务已创建"
}

# ============ 启动服务 ============
start_service() {
    if [[ "$SYSTEMD" == "false" ]]; then
        log_warn "systemd 不可用，无法启动服务"
        return
    fi

    log_step "启动服务..."

    systemctl enable node-push-exporter.service
    systemctl start node-push-exporter.service

    sleep 2

    if systemctl is-active --quiet node-push-exporter.service; then
        log_info "服务启动成功"
    else
        log_error "服务启动失败，查看日志: journalctl -u node-push-exporter -n 50"
        exit 1
    fi
}

# ============ 验证安装 ============
verify_installation() {
    log_step "验证安装..."

    # 检查二进制文件
    if [[ ! -x "$INSTALL_DIR/$BINARY_NAME" ]]; then
        log_error "node-push-exporter 未安装"
        exit 1
    fi

    if [[ ! -x "$INSTALL_DIR/$EXPORTER_NAME" ]]; then
        log_error "node_exporter 未安装"
        exit 1
    fi

    # 检查配置文件
    if [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
        log_error "配置文件不存在"
        exit 1
    fi

    # 检查服务状态
    if [[ "$SYSTEMD" == "true" ]]; then
        if systemctl is-enabled --quiet node-push-exporter.service; then
            log_info "开机自启: 已启用"
        fi
        if systemctl is-active --quiet node-push-exporter.service; then
            log_info "服务状态: 运行中"
        fi
    fi

    echo ""
    log_info "==================== 安装完成 ===================="
    log_info "node_exporter: $INSTALL_DIR/$EXPORTER_NAME"
    log_info "程序: $INSTALL_DIR/$BINARY_NAME"
    log_info "配置: $CONFIG_DIR/config.yaml"
    log_info "Pushgateway: $PUSHGATEWAY_URL"
    echo ""
    log_info "查看日志: journalctl -u node-push-exporter -f"
    log_info "查看状态: systemctl status node-push-exporter"
    log_info "停止服务: systemctl stop node-push-exporter"
}

# ============ 主函数 ============
main() {
    log_info "开始安装 node-push-exporter..."
    log_info "Pushgateway: $PUSHGATEWAY_URL"
    log_info "推送间隔: ${PUSH_INTERVAL}秒"
    log_info "任务名称: $JOB_NAME"

    # 检查 root 权限
    if [[ $EUID -ne 0 ]]; then
        log_error "请使用 root 权限运行 (sudo)"
        exit 1
    fi

    parse_args "$@"
    detect_os
    create_directories
    install_node_exporter
    install_exporter
    create_config
    create_systemd_service
    start_service
    verify_installation
}

main "$@"
