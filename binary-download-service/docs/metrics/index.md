---
title: 指标文档
description: Node Push Exporter 指标文档
---

# 指标文档

本节文档详细介绍 Node Push Exporter 采集的所有指标。

## 📊 指标类型

### 系统指标 (node_exporter)

Prometheus 官方 [node_exporter](https://github.com/prometheus/node_exporter) 采集的操作系统指标。

| 类别 | 说明 |
|:----:|------|
| 🖥️ CPU | 使用率、负载、上下文切换 |
| 💾 内存 | 总量、使用量、可用量、缓存 |
| 💿 磁盘 | 空间、I/O、读写速率 |
| 🌐 网络 | 流量、连接数、错误率 |
| 📁 文件系统 | 挂载点、inode、容量 |

### GPU 指标 (NVIDIA / AMD ROCm)

Node Push Exporter 额外采集的显卡指标。

| 类别 | 说明 |
|:----:|------|
| 🌡️ 温度 | 核心温度、结温、显存温度 |
| 📈 利用率 | GPU 计算利用率 |
| 🎮 显存 | 使用量、使用率、总量 |
| ⚡ 功耗 | 当前功耗、功率上限 |

## 🚀 快速查询

```promql
# 查看所有指标
{job="node"}

# CPU 使用率
(1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100

# GPU 利用率
gpu_utilization_percent{job="node"}
```

## 📚 文档导航

<div class="doc-cards">

| 文档 | 说明 |
|------|------|
| [Prometheus 查询示例](./prometheus) | 常用 PromQL 查询 + Python API |
| [Node Exporter 指标](./node-exporter) | 标准系统指标完整列表 |
| [GPU 指标](./gpu) | GPU 指标详解与采集原理 |

</div>

## 📋 标签说明

所有指标都带有以下标签：

| 标签 | 说明 | 示例 |
|------|------|------|
| `job` | Pushgateway job 名 | `node` |
| `instance` | 实例标识 (IP) | `10.17.154.252` |
| `vendor` | GPU 厂商 | `nvidia`, `rocm` |
| `gpu` | GPU 设备编号 | `0`, `1` |