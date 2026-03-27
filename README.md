# node-push-exporter

一款轻量级的系统指标推送工具，通过启动 node_exporter 采集系统指标，并推送到 Prometheus Pushgateway。

## 功能特性

- **使用 node_exporter**：通过官方 node_exporter 采集指标，支持多种操作系统
- **自动启动**：自动启动和管理 node_exporter 子进程
- **Pushgateway 推送**：定时将指标推送到 Prometheus Pushgateway
- **一键部署**：只需一条 curl 命令即可完成安装
- **开机自启**：集成 systemd 服务，支持系统启动自动运行

## 架构说明

```
┌─────────────────────────────────────────────────────────┐
│                    目标机器                              │
│                                                         │
│  ┌──────────────────┐      ┌──────────────────────┐   │
│  │  node_exporter   │      │  node-push-exporter  │   │
│  │  (子进程 :9100)  │─────▶│  (抓取 & 推送)       │   │
│  └──────────────────┘      └──────────┬───────────┘   │
│                                       │               │
│                              Pushgateway               │
└───────────────────────────────────────┼───────────────┘
                                        │
                                        ▼
                              ┌─────────────────────┐
                              │  Prometheus Server  │
                              └─────────────────────┘
```

## 快速安装

```bash
# 使用默认配置安装
curl -sL https://example.com/install.sh | sudo bash

# 指定 Pushgateway 地址安装
curl -sL https://example.com/install.sh | sudo bash -s -- --pushgateway http://prometheus:9091

# 完整参数安装示例
curl -sL https://example.com/install.sh | sudo bash -s -- \
  --pushgateway http://prometheus:9091 \
  --interval 30 \
  --job mynode \
  --instance server1
```

## 手动安装

### 1. 安装 node_exporter

```bash
# 下载 node_exporter
VERSION="1.8.1"
wget https://github.com/prometheus/node_exporter/releases/download/v${VERSION}/node_exporter-${VERSION}.linux-amd64.tar.gz

# 解压并安装
tar xzf node_exporter-${VERSION}.linux-amd64.tar.gz
sudo cp node_exporter-${VERSION}.linux-amd64/node_exporter /usr/local/bin/
sudo chmod +x /usr/local/bin/node_exporter
```

### 2. 安装 node-push-exporter

```bash
# 克隆项目并构建
git clone https://github.com/your-repo/node-push-exporter.git
cd node-push-exporter
go build -o node-push-exporter ./src/

# 安装
sudo cp node-push-exporter /usr/local/bin/
sudo chmod +x /usr/local/bin/node-push-exporter
```

### 3. 创建配置

本地直接运行时，程序默认读取当前目录下的 `./config.yml`。
如果使用 systemd 安装，再通过 `--config /etc/node-push-exporter/config.yaml` 显式指定系统配置路径。

```bash
vim ./config.yml

# 或安装到系统路径
sudo mkdir -p /etc/node-push-exporter
sudo cp ./config.yml /etc/node-push-exporter/config.yaml
sudo vim /etc/node-push-exporter/config.yaml
```

### 4. 创建 systemd 服务

```bash
sudo cp systemd/node-push-exporter.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable node-push-exporter
sudo systemctl start node-push-exporter
```

## 配置说明

默认配置文件路径为当前目录下的 `./config.yml`。
如果通过 systemd 安装，服务文件会显式读取 `/etc/node-push-exporter/config.yaml`。
配置格式为 `key=value`：

```ini
# Pushgateway 设置
pushgateway.url=http://localhost:9091     # Pushgateway 地址
pushgateway.job=node                      # 任务名称
pushgateway.instance=                     # 实例名称(可选，默认使用主机名)
pushgateway.interval=60                   # 推送间隔(秒)
pushgateway.timeout=10                    # HTTP 超时(秒)

# node_exporter 设置
node_exporter.path=/usr/local/bin/node_exporter  # node_exporter 路径
node_exporter.port=9100                           # 监听端口
node_exporter.metrics_url=http://localhost:9100/metrics  # 抓取地址
```

## 支持的指标

node_exporter 采集的完整指标，包括但不限于：

| 类别 | 指标前缀 | 说明 |
|------|----------|------|
| CPU | `node_cpu_*` | CPU 使用率、时间等 |
| 内存 | `node_memory_*` | 内存总量、使用量、空闲等 |
| 磁盘 | `node_disk_*` | 磁盘空间、IO 等 |
| 网络 | `node_network_*` | 网络流量、错误等 |
| 文件系统 | `node_filesystem_*` | 文件系统使用情况 |
| 负载 | `node_load*` | 系统负载 |
| 时间 | `node_time_*` | 系统时间 |

完整指标列表请参考 [node_exporter README](https://github.com/prometheus/node_exporter)。

## 常用命令

```bash
# 查看运行日志
sudo journalctl -u node-push-exporter -f

# 停止服务
sudo systemctl stop node-push-exporter

# 启动服务
sudo systemctl start node-push-exporter

# 重启服务
sudo systemctl restart node-push-exporter

# 查看服务状态
sudo systemctl status node-push-exporter

# 查看版本
node-push-exporter --version

# 直接访问 node_exporter 指标
curl http://localhost:9100/metrics
```

## 卸载

```bash
sudo systemctl stop node-push-exporter
sudo systemctl disable node-push-exporter
sudo rm /etc/systemd/system/node-push-exporter.service
sudo rm /usr/local/bin/node-push-exporter
sudo rm /usr/local/bin/node_exporter
sudo rm -rf /etc/node-push-exporter
```

## 故障排查

### 问题：服务启动失败

```bash
# 查看详细日志
journalctl -u node-push-exporter -n 100

# 检查 node_exporter 是否可以运行
/usr/local/bin/node_exporter --version
```

### 问题：无法连接到 Pushgateway

```bash
# 测试网络连接
curl -v http://your-pushgateway:9091/metrics
```

## 许可证

MIT
