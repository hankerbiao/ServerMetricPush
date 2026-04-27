---
title: Python API 查询
description: 使用 Python 查询 Prometheus 指标数据
---

# Python API 查询

使用 `prometheus-api-client` 库查询 Prometheus 指标数据。

## 安装

```bash
pip install prometheus-api-client
```

## 基础查询

### 连接 Prometheus

```python
from prometheus_api_client import PrometheusConnect

# 连接 Prometheus
prom = PrometheusConnect(url="http://localhost:9090", disable_ssl=True)

# 带认证连接
prom = PrometheusConnect(
    url="http://prometheus.example.com:9090",
    headers={"Authorization": "Bearer your-token"}
)
```

### 查询 CPU 使用率

```python
# 查询所有节点 CPU 使用率
cpu_query = '''
(1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100
'''
cpu_data = prom.custom_query(query=cpu_query)

for metric in cpu_data:
    instance = metric["metric"]["instance"]
    value = float(metric["value"][1])
    print(f"{instance}: {value:.2f}%")
```

### 查询结果格式

```python
# 返回格式
[
    {
        "metric": {"instance": "192.168.1.100", "job": "node"},
        "value": [1704067200.0, "23.45"]
    }
]

# 解析
for metric in data:
    labels = metric["metric"]           # 标签字典
    timestamp = metric["value"][0]      # Unix 时间戳
    value = float(metric["value"][1])   # 指标值
```

---

## 系统指标查询

### 内存使用率

```python
mem_query = '''
(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100
'''
mem_data = prom.custom_query(query=mem_query)

for metric in mem_data:
    instance = metric["metric"]["instance"]
    mem_pct = float(metric["value"][1])
    print(f"{instance}: 内存 {mem_pct:.1f}%")
```

### 磁盘使用率

```python
disk_query = '''
(node_filesystem_size_bytes - node_filesystem_avail_bytes) / node_filesystem_size_bytes * 100
'''
disk_data = prom.custom_query(query=disk_query)
```

### 网络流量

```python
# 接收速率 (MB/s)
recv_query = 'rate(node_network_receive_bytes_total{job="node"}[5m]) / 1024 / 1024'

# 发送速率 (MB/s)
send_query = 'rate(node_network_transmit_bytes_total{job="node"}[5m]) / 1024 / 1024'
```

---

## GPU 指标查询

### 查询所有 GPU 指标

```python
# GPU 利用率
gpu_util = prom.custom_query('gpu_utilization_percent{job="node"}')

# GPU 温度
gpu_temp = prom.custom_query('gpu_temperature_celsius{job="node"}')

# GPU 显存
gpu_mem = prom.custom_query('gpu_memory_used_percent{job="node"}')

# GPU 功耗
gpu_power = prom.custom_query('gpu_power_draw_watts{job="node"}')
```

### 解析 GPU 数据

```python
for metric in gpu_util:
    instance = metric["metric"]["instance"]
    gpu_idx = metric["metric"]["gpu"]
    vendor = metric["metric"]["vendor"]
    util = float(metric["value"][1])
    print(f"{instance} GPU {gpu_idx} ({vendor}): {util:.1f}%")
```

### 按厂商过滤

```python
# NVIDIA GPU
nvidia_util = prom.custom_query('gpu_utilization_percent{job="node", vendor="nvidia"}')

# AMD ROCm GPU
amd_util = prom.custom_query('gpu_utilization_percent{job="node", vendor="rocm"}')
```

### 温度告警检测

```python
high_temp = prom.custom_query('gpu_temperature_celsius{job="node"} > 80')

if high_temp:
    print(f"⚠️  {len(high_temp)} 个 GPU 温度过高!")
    for m in high_temp:
        print(f"  {m['metric']['instance']} GPU {m['metric']['gpu']}: {m['value'][1]}°C")
```

---

## 高级用法

### 按实例过滤

```python
target_ip = "192.168.1.100"

# 查询特定实例
query = '''
(1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100
'''
results = prom.custom_query(query=query)

filtered = [
    m for m in results
    if m["metric"].get("instance") == target_ip
]
```

### 批量查询

```python
import concurrent.futures

def query_metric(prom, query):
    return query, prom.custom_query(query)

queries = {
    "cpu_usage": '(1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100',
    "memory_usage": '(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100',
    "gpu_util": 'gpu_utilization_percent{job="node"}',
    "gpu_temp": 'avg by (instance) (gpu_temperature_celsius{job="node"})',
}

with concurrent.futures.ThreadPoolExecutor(max_workers=4) as executor:
    futures = {
        executor.submit(query_metric, prom, q): name
        for name, q in queries.items()
    }
    for future in concurrent.futures.as_completed(futures):
        name = futures[future]
        query, results = future.result()
        print(f"\n{name}: {len(results)} results")
        for r in results[:3]:  # 只打印前3个
            print(f"  {r['metric']['instance']}: {float(r['value'][1]):.2f}")
```

### 查询节点元数据

```python
# 所有注册的节点
nodes = prom.custom_query('count by (instance) (node_cpu_seconds_total{job="node"})')

# GPU 数量统计
gpu_count = prom.custom_query('sum by (instance) (node_push_exporter_gpu_devices_detected{job="node"})')

# 总功耗
total_power = prom.custom_query('sum by (instance) (gpu_power_draw_watts{job="node"})')
```

### 完整示例：生成监控报告

```python
from prometheus_api_client import PrometheusConnect
from datetime import datetime

def generate_report(prom, instance=None):
    """生成节点监控报告"""
    filters = f',instance="{instance}"' if instance else ''

    queries = {
        "CPU": f'avg by (instance) (rate(node_cpu_seconds_total{{mode="idle"{filters}}}[5m])) * 100',
        "内存": f'(1 - (node_memory_MemAvailable_bytes{filters} / node_memory_MemTotal_bytes{filters})) * 100',
        "GPU利用率": f'avg by (instance) (gpu_utilization_percent{{job="node"{filters}}})',
        "GPU温度": f'avg by (instance) (gpu_temperature_celsius{{job="node"{filters}}})',
    }

    report = {}
    for name, query in queries.items():
        try:
            data = prom.custom_query(query)
            values = {m["metric"]["instance"]: float(m["value"][1]) for m in data}
            report[name] = values
        except Exception as e:
            report[name] = {"error": str(e)}

    return report

# 使用
prom = PrometheusConnect(url="http://localhost:9090", disable_ssl=True)
report = generate_report(prom, instance="192.168.1.100")

print("=== 监控报告 ===")
print(f"生成时间: {datetime.now()}")
for metric, values in report.items():
    print(f"\n{metric}:")
    for node, val in values.items():
        print(f"  {node}: {val:.1f}")
```

---

## 完整查询示例

```python
from prometheus_api_client import PrometheusConnect

# 初始化
prom = PrometheusConnect(url="http://localhost:9090", disable_ssl=True)

# 查询所有节点状态
all_nodes = prom.custom_query('count by (instance) (node_cpu_seconds_total{job="node"})')
print(f"节点数量: {len(all_nodes)}")

# 查询 GPU 节点
gpu_nodes = prom.custom_query('count by (instance) (node_push_exporter_gpu_devices_detected{job="node"} > 0)')
print(f"GPU 节点数量: {len(gpu_nodes)}")

# 性能排行榜
top_cpu = prom.custom_query('''
topk(10, (1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100)
''')

top_mem = prom.custom_query('''
topk(10, (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100)
''')

print(f"CPU 使用率最高的 10 台机器:")
for m in top_cpu:
    print(f"  {m['metric']['instance']}: {float(m['value'][1]):.1f}%")
```