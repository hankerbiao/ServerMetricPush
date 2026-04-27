---
title: 概述
description: Node Push Exporter 项目概述
---

# 概述

## 项目简介

Node Push Exporter 是一个用于采集系统指标并推送到 Prometheus Pushgateway 的轻量级服务。它作为 `node_exporter` 的包装器，自动管理 `node_exporter` 子进程，并按配置的间隔将采集的指标推送到 Pushgateway。

## 核心功能

### 1. 指标采集与推送

- **node_exporter 指标**: 自动采集 CPU、内存、磁盘、网络等标准系统指标
- **自定义 GPU 指标**: 支持 NVIDIA 和 AMD ROCm GPU 的温度、功耗、显存、利用率等指标
- **定时推送**: 可配置的推送间隔（默认 30 秒），自动推送到 Pushgateway

### 2. 节点管理

- **自动注册**: 启动时自动向管理服务注册节点信息
- **心跳机制**: 定期发送心跳，报告节点状态和推送结果
- **状态监控**: 支持在线 (online)、降级 (degraded)、离线 (offline) 三种状态

### 3. 二进制分发服务

配套的 **Binary Download Service** 提供：

- 二进制文件上传、存储、分发
- 多平台、多架构版本管理
- Web 管理界面
- RESTful API

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      Prometheus/Grafana                      │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ Pull
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Prometheus Pushgateway                    │
│                    (接收来自各节点的推送)                      │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ Push
┌─────────────────────────────────────────────────────────────┐
│                  node-push-exporter (每节点)                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   指标采集    │  │   指标推送    │  │   GPU 指标采集      │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│         │                                          │         │
│         ▼                                          ▼         │
│  ┌─────────────┐                          ┌─────────────────┐│
│  │node_exporter│                          │nvidia-smi/rocm-smi│
│  └─────────────┘                          └─────────────────┘│
└─────────────────────────────────────────────────────────────┘
                              │
                              │ Register/Heartbeat
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Binary Download Service                    │
│                   (节点注册与监控管理)                         │
└─────────────────────────────────────────────────────────────┘
```

## 支持的指标

### 系统指标 (node_exporter)

| 类别 | 指标示例 |
|------|---------|
| CPU | `node_cpu_seconds_total`, `node_load1` |
| 内存 | `node_memory_MemTotal_bytes`, `node_memory_MemAvailable_bytes` |
| 磁盘 | `node_disk_read_bytes_total`, `node_filesystem_avail_bytes` |
| 网络 | `node_network_receive_bytes_total`, `node_network_transmit_bytes_total` |

### 自定义 GPU 指标

| 类别 | 指标名称 | 说明 |
|------|---------|------|
| 采集状态 | `node_push_exporter_gpu_scrape_success` | GPU 采集是否成功 |
| 设备数量 | `node_push_exporter_gpu_devices_detected` | 检测到的 GPU 设备数 |
| 温度 | `gpu_temperature_celsius` | GPU 温度（℃） |
| 利用率 | `gpu_utilization_percent` | GPU 利用率（%） |
| 显存 | `gpu_memory_used_percent`, `gpu_memory_used_bytes` | 显存使用情况 |
| 功耗 | `gpu_power_draw_watts` | GPU 功耗（W） |

## 技术栈

| 组件 | 技术 |
|------|------|
| 主程序 | Go 1.21+ |
| 指标采集 | node_exporter, nvidia-smi, rocm-smi |
| 指标推送 | Prometheus Pushgateway |
| 管理服务 | FastAPI + SQLite |
| 文档站点 | VitePress |
