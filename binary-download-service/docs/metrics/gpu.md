---
title: 自定义 GPU 指标
description: Node Push Exporter 自定义 GPU 指标详解
---

# 自定义 GPU 指标

Node Push Exporter 通过调用 `nvidia-smi` 和 `rocm-smi` 采集额外的 GPU 指标，这些指标不包含在标准 node_exporter 中。

## 指标采集状态

### node_push_exporter_gpu_scrape_timestamp_seconds

GPU 指标采集时间戳。

**类型:** Gauge

**标签:** 无

### node_push_exporter_gpu_scrape_success

GPU 指标采集是否成功。

**类型:** Gauge

**标签:** `vendor`

| 值 | 含义 |
|----|------|
| 1 | 采集成功 |
| 0 | 采集失败 |

### node_push_exporter_gpu_devices_detected

检测到的 GPU 设备数量。

**类型:** Gauge

**标签:** `vendor`

```promql
# 检查 GPU 是否被正确识别
node_push_exporter_gpu_devices_detected{job="node"}

# 按厂商统计 GPU 数量
sum by (vendor) (node_push_exporter_gpu_devices_detected{job="node"})
```

## 通用 GPU 指标

以下指标同时适用于 NVIDIA 和 AMD ROCm GPU：

### gpu_up

GPU 设备是否在线。

**类型:** Gauge

**标签:** `vendor`, `gpu`, `name`, `uuid`, `device_id`

| 值 | 含义 |
|----|------|
| 1 | GPU 在线 |
| 0 | GPU 离线 |

### gpu_info

GPU 设备信息（恒定为 1）。

**类型:** Gauge

**标签:** `vendor`, `gpu`, `name`, `uuid`, `device_id`

用于在查询时获取 GPU 静态信息。

## 温度指标

### gpu_temperature_celsius

GPU 核心温度。

**类型:** Gauge

**标签:** `vendor`, `gpu`, `name`, `uuid`, `device_id`

```promql
# 获取所有 GPU 温度
gpu_temperature_celsius{job="node"}

# 按实例统计平均温度
avg by (instance) (gpu_temperature_celsius{job="node"})
```

### gpu_temperature_edge_celsius

GPU Edge 温度传感器读数。

**类型:** Gauge

**标签:** 同上

### gpu_temperature_junction_celsius

GPU 结温（Junction Temperature）。

**类型:** Gauge

**标签:** 同上

### gpu_temperature_mem_celsius

GPU 显存温度。

**类型:** Gauge

**标签:** 同上

### gpu_temperature_core_celsius

GPU 核心温度（备选传感器）。

**类型:** Gauge

**标签:** 同上

### 温度告警阈值

| 级别 | 阈值 | 建议操作 |
|------|------|----------|
| 正常 | < 70°C | 无需处理 |
| 警告 | 70-85°C | 监控频率增加 |
| 危险 | > 85°C | 检查散热 |
| 临界 | > 95°C | 可能触发降频保护 |

## 利用率指标

### gpu_utilization_percent

GPU 计算利用率。

**类型:** Gauge

**标签:** `vendor`, `gpu`, `name`, `uuid`, `device_id`

**范围:** 0-100

```promql
# GPU 利用率
gpu_utilization_percent{job="node"}

# 利用率超过 90% 的 GPU
gpu_utilization_percent{job="node"} > 90

# 按实例统计平均利用率
avg by (instance) (gpu_utilization_percent{job="node"})
```

## 显存指标

### gpu_memory_used_percent

GPU 显存使用率。

**类型:** Gauge

**标签:** `vendor`, `gpu`, `name`, `uuid`, `device_id`

**范围:** 0-100

```promql
# 显存使用率
gpu_memory_used_percent{job="node"}

# 显存使用率超过 80%
gpu_memory_used_percent{job="node"} > 80
```

### gpu_memory_used_bytes

已使用的 GPU 显存（字节）。

**类型:** Gauge

**标签:** 同上

```promql
# 已使用显存 (GB)
gpu_memory_used_bytes{job="node"} / 1024 / 1024 / 1024
```

### gpu_memory_total_bytes

GPU 显存总量（字节）。

**类型:** Gauge

**标签:** 同上

```promql
# 总显存 (GB)
gpu_memory_total_bytes{job="node"} / 1024 / 1024 / 1024
```

## 功耗指标

### gpu_power_draw_watts

GPU 当前功耗。

**类型:** Gauge

**标签:** `vendor`, `gpu`, `name`, `uuid`, `device_id`

**单位:** 瓦特 (W)

```promql
# GPU 当前功耗
gpu_power_draw_watts{job="node"}

# 单实例 GPU 总功耗
sum by (instance) (gpu_power_draw_watts{job="node"})

# 单实例平均单卡功耗
avg by (instance) (gpu_power_draw_watts{job="node"})
```

## NVIDIA 特有信息

NVIDIA GPU 会通过 `nvidia-smi` 获取以下额外信息：

| 字段 | 说明 | 示例 |
|------|------|------|
| `gpu` | GPU 编号 | `0`, `1` |
| `name` | GPU 型号名称 | `NVIDIA GeForce RTX 3090` |
| `uuid` | GPU 唯一标识符 | `GPU-xxxx-xxxx-xxxx-xxxx` |

## AMD ROCm 特有信息

AMD GPU 会通过 `rocm-smi` 获取以下额外信息：

| 字段 | 说明 | 示例 |
|------|------|------|
| `gpu` | GPU 编号 | `0`, `1` |
| `device_id` | 设备 ID | `card0`, `card1` |
| `name` | GPU 型号名称 | ` Vega 10 [Radeon RX Vega 56/64]` |

## 查询示例

### 完整 GPU 状态仪表盘

```promql
# GPU 基本信息
gpu_info{job="node"}

# 温度 (着色显示)
gpu_temperature_celsius{job="node"}

# 利用率 (百分比)
gpu_utilization_percent{job="node"}

# 显存使用率
gpu_memory_used_percent{job="node"}

# 功耗
gpu_power_draw_watts{job="node"}
```

### 按实例汇总

```promql
# 每台服务器的 GPU 数量
sum by (instance) (node_push_exporter_gpu_devices_detected{job="node"})

# 每台服务器的平均 GPU 温度
avg by (instance) (gpu_temperature_celsius{job="node"})

# 每台服务器的平均 GPU 利用率
avg by (instance) (gpu_utilization_percent{job="node"})

# 每台服务器的总 GPU 功耗
sum by (instance) (gpu_power_draw_watts{job="node"})

# 每台服务器的总显存使用
sum by (instance) (gpu_memory_used_bytes{job="node"}) / 1024 / 1024 / 1024
```

### 按厂商分析

```promql
# NVIDIA vs AMD GPU 分布
count by (vendor) (gpu_info{job="node"})

# 各厂商平均温度
avg by (vendor) (gpu_temperature_celsius{job="node"})

# 各厂商平均利用率
avg by (vendor) (gpu_utilization_percent{job="node"})
```

## 故障排查

### GPU 指标采集失败

检查 `node_push_exporter_gpu_scrape_success`:

```promql
# 找出采集失败的 GPU
node_push_exporter_gpu_scrape_success{job="node"} == 0
```

可能原因：
1. 未安装对应驱动
2. nvidia-smi 或 rocm-smi 不在 PATH 中
3. 无 GPU 设备
4. 权限不足

### GPU 设备数为 0

```promql
# 检查是否有设备
node_push_exporter_gpu_devices_detected{job="node"} == 0
```

可能原因：
1. 系统没有 GPU
2. GPU 驱动未加载
3. 虚拟化环境未穿透 GPU