# node-push-exporter

[![MIT License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

一个轻量级的 Prometheus 指标推送服务，自动启动 `node_exporter` 并将系统指标推送到 Pushgateway。

## 特性

- **零依赖运行**: 自动管理 `node_exporter` 子进程，无需额外部署
- **GPU 监控**: 支持 NVIDIA GPU 和 AMD GPU 指标采集
- **节点注册**: 可选地向控制面服务注册，实现集中管理
- **多平台支持**: 支持 Linux AMD64 / ARM64 / ARMv7

## 快速开始

### 安装

```bash
# 一键安装（自动检测平台并安装）
curl -fsSL https://your-binary-server/download/node-push-exporter-linux-$(uname -m).tar.gz | sudo bash
```

### 配置

编辑 `config.yml`:

```ini
# Pushgateway 设置
pushgateway.url=http://your-prometheus:9091
pushgateway.job=node
pushgateway.instance=                    # 自动使用本机 IP
pushgateway.interval=60
pushgateway.timeout=10

# node_exporter 设置
node_exporter.path=node_exporter
node_exporter.port=9100
node_exporter.metrics_url=http://localhost:9100/metrics

# 控制面设置（可选）
control_plane.url=http://your-server:8080
control_plane.heartbeat_interval=30
```

### 运行

```bash
./node-push-exporter
```

## 采集指标

### 系统指标 (node_exporter)

| 类别 | 指标 |
|------|------|
| CPU | 使用率、负载 |
| 内存 | 总量、使用量、可用量 |
| 磁盘 | 分区空间、IO 操作 |
| 网络 | 网卡流量、连接状态 |
| 文件系统 | 挂载点容量 |

### GPU 指标

| 指标 | 说明 |
|------|------|
| `gpu_up` | GPU 是否在线 |
| `gpu_utilization_percent` | GPU 利用率 |
| `gpu_memory_used_percent` | 显存使用率 |
| `gpu_power_draw_watts` | 功耗 |
| `gpu_temperature_*` | 温度 |

GPU 采集使用 `nvidia-smi` (NVIDIA) 或 `rocm-smi` (AMD)。

## Prometheus 查询示例

```promql
# CPU 使用率
100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)

# 内存使用率
100 - (avg by (instance) (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100)

# 磁盘使用率
100 - (avg by (instance, device) (node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100)

# 网络流量
rate(node_network_receive_bytes_total[5m])
rate(node_network_transmit_bytes_total[5m])

# GPU 利用率
gpu_utilization_percent

# GPU 显存使用率
gpu_memory_used_percent

# GPU 温度
gpu_temperature_edge
```

## 节点管理

启用控制面后，可在 Web UI 查看节点状态：

```
http://your-server:8080/agents
```

功能包括：
- 节点在线/离线状态
- 最近心跳时间
- Pushgateway 推送结果
- 错误信息汇总

## 构建

```bash
# 编译所有平台
./build.sh

# 开发模式（构建但不上传）
./build.sh -d
```

产物位于 `./releases/` 目录。

## 项目结构

```
node-push-exporter/
├── src/
│   ├── main.go           # 入口点
│   ├── config/           # 配置加载
│   ├── process/          # node_exporter 进程管理
│   ├── exporter/         # 指标采集和推送
│   ├── pusher/           # Pushgateway 客户端
│   ├── controlplane/     # 控制面通信
│   ├── gpu/              # GPU 指标采集
│   ├── runtime/          # 运行时状态
│   └── metrics/          # 内部指标
├── config.yml            # 配置文件示例
├── build.sh              # 构建脚本
├── systemd/              # systemd 服务文件
└── binary-download-service/  # 二进制分发服务
```

## 故障排查

```bash
# 查看运行状态
sudo systemctl status node-push-exporter

# 查看日志
sudo journalctl -u node-push-exporter -f

# 验证指标
curl http://localhost:9100/metrics | head
```

## License

MIT