# node-push-exporter 安装部署文档

本文档介绍 `node-push-exporter` 的安装和部署方法。

## 系统要求

- **操作系统**: Linux (支持 systemd)
- **架构**: amd64, arm64, armv7
- **依赖**: node_exporter 二进制文件
- **权限**: root 或 sudo 权限

## 快速安装 (推荐)

使用一键安装脚本：

```bash
curl -fsSL http://10.17.154.252:8888/download/install.sh | sudo bash
```

安装脚本会自动完成：
1. 根据系统架构下载 node_exporter 和 node-push-exporter
2. 安装到 `/usr/local/bin/`
3. 创建配置目录 `/etc/node-push-exporter/`
4. 配置 systemd 服务
5. 启动服务并设置开机自启

## 手动安装

### 步骤 1: 下载二进制文件

从 releases 目录或构建服务器下载对应架构的版本：

```bash
# amd64
curl -fsSL http://10.17.154.252:8888/download/node-push-exporter-linux-amd64 -o /usr/local/bin/node-push-exporter

# arm64
curl -fsSL http://10.17.154.252:8888/download/node-push-exporter-linux-arm64 -o /usr/local/bin/node-push-exporter

# armv7
curl -fsSL http://10.17.154.252:8888/download/node-push-exporter-linux-armv7 -o /usr/local/bin/node-push-exporter
```

设置执行权限：

```bash
chmod +x /usr/local/bin/node-push-exporter
```

### 步骤 2: 下载 node_exporter

从 Prometheus 官网下载 node_exporter：

```bash
# amd64
curl -fsSL https://github.com/prometheus/node_exporter/releases/download/v1.7.0/node_exporter-1.7.0.linux-amd64.tar.gz -o /tmp/node_exporter.tar.gz

# 解压
tar -xzf /tmp/node_exporter.tar.gz -C /tmp/
cp /tmp/node_exporter-1.7.0.linux-amd64/node_exporter /usr/local/bin/node_exporter
chmod +x /usr/local/bin/node_exporter

# 清理
rm -rf /tmp/node_exporter*
```

### 步骤 3: 创建配置文件

创建配置目录和配置文件：

```bash
mkdir -p /etc/node-push-exporter
```

创建 `config.yml` 文件，配置格式为 key=value：

```bash
# Pushgateway 设置
pushgateway.url=http://localhost:9091
pushgateway.job=node
pushgateway.instance=
pushgateway.interval=60
pushgateway.timeout=10

# node_exporter 设置
node_exporter.path=node_exporter
node_exporter.port=9100
node_exporter.metrics_url=http://localhost:9100/metrics

# 控制面设置 (可选)
# control_plane.url=http://localhost:8888
# control_plane.heartbeat_interval=30
```

### 步骤 4: 配置 systemd 服务

将服务文件复制到系统目录：

```bash
cp systemd/node-push-exporter.service /etc/systemd/system/
systemctl daemon-reload
```

### 步骤 5: 启动服务

```bash
# 启动服务
systemctl start node-push-exporter

# 设置开机自启
systemctl enable node-push-exporter

# 查看服务状态
systemctl status node-push-exporter
```

## 配置说明

### Pushgateway 配置

| 配置项 | 必填 | 说明 | 示例 |
|--------|------|------|------|
| pushgateway.url | 是 | Pushgateway 地址 | http://10.17.154.252:9091 |
| pushgateway.job | 是 | 推送的 job 名称 | node |
| pushgateway.instance | 否 | 实例标识，留空则自动使用本机 IP | - |
| pushgateway.interval | 是 | 推送间隔(秒) | 60 |
| pushgateway.timeout | 是 | HTTP 请求超时(秒) | 10 |

### node_exporter 配置

| 配置项 | 必填 | 说明 | 示例 |
|--------|------|------|------|
| node_exporter.path | 是 | node_exporter 可执行文件路径 | node_exporter |
| node_exporter.port | 是 | node_exporter 监听端口 | 9100 |
| node_exporter.metrics_url | 是 | 抓取指标的完整 URL | http://localhost:9100/metrics |

### 控制面配置 (可选)

| 配置项 | 必填 | 说明 | 示例 |
|--------|------|------|------|
| control_plane.url | 是* | 控制面服务地址 | http://localhost:8888 |
| control_plane.heartbeat_interval | 是* | 心跳间隔(秒) | 30 |

*当其中一个配置项存在时，另一个也必须配置。

## 验证部署

### 检查服务状态

```bash
systemctl status node-push-exporter --no-pager
```

### 检查指标采集

```bash
# 检查 node_exporter 指标
curl http://127.0.0.1:9100/metrics | head -20

# 检查推送日志
journalctl -u node-push-exporter -f --no-pager
```

### 验证 Pushgateway 接收

访问 Pushgateway 的 Web 界面，确认指标已接收：
```
http://<pushgateway地址>/metrics
```

## 卸载

使用一键卸载脚本：

```bash
curl -fsSL http://10.17.154.252:8888/download/uninstall.sh | sudo bash
```

或手动卸载：

```bash
# 停止并禁用服务
systemctl stop node-push-exporter
systemctl disable node-push-exporter

# 删除文件
rm -f /usr/local/bin/node-push-exporter
rm -rf /etc/node-push-exporter
rm -f /etc/systemd/system/node-push-exporter.service

# 重载 systemd
systemctl daemon-reload
```

## 构建源码

如需从源码构建：

```bash
# 克隆项目
git clone <repository-url>
cd push_node

# 构建
go build -o node-push-exporter ./src

# 查看版本
./node-push-exporter --version
```

## 故障排查

### 服务启动失败

检查日志：
```bash
journalctl -u node-push-exporter -n 100 --no-pager
```

常见问题：
- 配置文件格式错误 (必须是 key=value 格式)
- node_exporter 路径不正确
- Pushgateway 地址不可达

### 指标未推送

1. 确认 Pushgateway 服务正常运行
2. 检查网络连通性：`telnet <pushgateway-host> 9091`
3. 检查防火墙是否放行相应端口