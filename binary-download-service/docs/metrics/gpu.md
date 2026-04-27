---
title: GPU 指标
description: Node Push Exporter GPU 指标详解
---

# GPU 指标

Node Push Exporter 通过 `nvidia-smi` 和 `rocm-smi` 采集 GPU 指标。

## 📊 采集状态

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `node_push_exporter_gpu_scrape_success` | Gauge | 采集是否成功 (1=成功, 0=失败) |
| `node_push_exporter_gpu_devices_detected` | Gauge | 检测到的 GPU 数量 |

```promql
# 检查 GPU 是否正确识别
node_push_exporter_gpu_devices_detected{job="node"}

# 按厂商统计 GPU 数量
sum by (vendor) (node_push_exporter_gpu_devices_detected{job="node"})
```

---

## 🌡️ 温度指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `gpu_temperature_celsius` | Gauge | GPU 温度 (通用) |
| `gpu_temperature_edge_celsius` | Gauge | Edge 温度传感器 |
| `gpu_temperature_junction_celsius` | Gauge | 结温 (Junction) |
| `gpu_temperature_mem_celsius` | Gauge | 显存温度 |
| `gpu_temperature_core_celsius` | Gauge | 核心温度 (备选) |

### 温度告警阈值

| 状态 | 温度 | 建议 |
|:----:|:----:|------|
| ✅ 正常 | < 70°C | 无需处理 |
| ⚠️ 警告 | 70-85°C | 监控频率增加 |
| 🔥 危险 | 85-95°C | 检查散热 |
| 💀 临界 | > 95°C | 可能触发降频保护 |

```promql
# 获取所有 GPU 温度
gpu_temperature_celsius{job="node"}

# 平均温度
avg by (instance) (gpu_temperature_celsius{job="node"})

# 温度超过 80°C 的 GPU
gpu_temperature_celsius{job="node"} > 80
```

---

## 📈 GPU 利用率

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `gpu_utilization_percent` | Gauge | GPU 计算利用率 (%) |

```promql
# GPU 利用率
gpu_utilization_percent{job="node"}

# 利用率超过 90%
gpu_utilization_percent{job="node"} > 90

# 平均利用率
avg by (instance) (gpu_utilization_percent{job="node"})
```

---

## 🎮 显存指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `gpu_memory_used_percent` | Gauge | 显存使用率 (%) |
| `gpu_memory_used_bytes` | Gauge | 已使用显存 (bytes) |
| `gpu_memory_total_bytes` | Gauge | 显存总量 (bytes) |

```promql
# 显存使用率
gpu_memory_used_percent{job="node"}

# 已用显存 (GB)
gpu_memory_used_bytes{job="node"} / 1024^3

# 总显存 (GB)
gpu_memory_total_bytes{job="node"} / 1024^3

# 使用率超过 80%
gpu_memory_used_percent{job="node"} > 80
```

---

## ⚡ 功耗指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `gpu_power_draw_watts` | Gauge | 当前功耗 (W) |
| `gpu_power_limit_watts` | Gauge | 功率上限 (W) |

```promql
# 当前功耗
gpu_power_draw_watts{job="node"}

# 单机总功耗
sum by (instance) (gpu_power_draw_watts{job="node"})

# 单卡平均功耗
avg by (instance) (gpu_power_draw_watts{job="node"})
```

---

## 🌀 其他指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `gpu_fan_speed_percent` | Gauge | 风扇转速 (%) |
| `gpu_clock_sm_hertz` | Gauge | GPU SM 频率 (Hz) |
| `gpu_clock_mem_hertz` | Gauge | 显存频率 (Hz) |

---

## 🏷️ 标签说明

| 标签 | 说明 | 示例 |
|------|------|------|
| `vendor` | GPU 厂商 | `nvidia`, `rocm` |
| `gpu` | GPU 编号 | `0`, `1` |
| `name` | GPU 型号 | `NVIDIA GeForce RTX 3090` |
| `uuid` | GPU 唯一标识 | `GPU-xxxx-xxxx-xxxx-xxxx` |
| `device_id` | 设备 ID (AMD) | `card0` |

---

## 🔍 查询示例

### GPU 状态总览

```promql
# 所有 GPU 基本信息
gpu_info{job="node"}

# 温度 + 利用率 + 显存
gpu_temperature_celsius{job="node"}
gpu_utilization_percent{job="node"}
gpu_memory_used_percent{job="node"}
gpu_power_draw_watts{job="node"}
```

### 按实例汇总

```promql
# GPU 数量
sum by (instance) (node_push_exporter_gpu_devices_detected{job="node"})

# 平均温度 / 利用率 / 功耗
avg by (instance) (gpu_temperature_celsius{job="node"})
avg by (instance) (gpu_utilization_percent{job="node"})
sum by (instance) (gpu_power_draw_watts{job="node"})

# 总显存使用 (GB)
sum by (instance) (gpu_memory_used_bytes{job="node"}) / 1024^3
```

### 按厂商分析

```promql
# 各厂商 GPU 数量
count by (vendor) (gpu_info{job="node"})

# 各厂商平均利用率
avg by (vendor) (gpu_utilization_percent{job="node"})
```

---

## 🔧 故障排查

### GPU 采集失败

```promql
# 找出采集失败的实例
node_push_exporter_gpu_scrape_success{job="node"} == 0
```

可能原因：
- ❌ 未安装 GPU 驱动
- ❌ `nvidia-smi` 或 `rocm-smi` 不在 PATH
- ❌ 无 GPU 设备
- ❌ 权限不足

### 检测到 0 个 GPU

```promql
# 检查设备数量
node_push_exporter_gpu_devices_detected{job="node"} == 0
```

可能原因：
- 系统没有 GPU
- GPU 驱动未加载
- 虚拟化环境未穿透 GPU

### 手动验证

```bash
# NVIDIA
nvidia-smi

# AMD ROCm
rocm-smi
```