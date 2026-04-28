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
prom = PrometheusConnect(url="http://10.17.151.170:9090", disable_ssl=True)

# 带认证连接
prom = PrometheusConnect(
    url="http://10.17.151.170:9090",
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
        "metric": {"instance": "10.17.150.235", "job": "node"},
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
target_ip = "10.17.150.235"

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
import json
from prometheus_api_client import PrometheusConnect
from datetime import datetime

# 配置信息
PROM_URL = "http://10.17.154.252:9090"
ALIVE_THRESHOLD = 600

prom = PrometheusConnect(url=PROM_URL, disable_ssl=True)


def get_node_metrics_json():
    """
    执行 Prometheus 查询并返回格式化的 JSON 数据
    """
    # 统一定义查询语句（Key 为 JSON 中的指标名）
    queries = {
        "cpu_usage_percent": f'clamp_max(avg by (instance) (rate(node_cpu_seconds_total{{mode!~"idle|iowait"}}[5m])) * 100, 100) and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
        "memory_usage_percent": f'(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
        "network_throughput_mbps": f'sum by (instance) (rate(node_network_receive_bytes_total{{device!~"lo|docker.*|veth.*"}}[5m]) + rate(node_network_transmit_bytes_total{{device!~"lo|docker.*|veth.*"}}[5m])) / 1024 / 1024 and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
        "disk_io_mbps": f'sum by (instance) (rate(node_disk_read_bytes_total{{device!~"loop.*|ram.*"}}[5m]) + rate(node_disk_written_bytes_total{{device!~"loop.*|ram.*"}}[5m])) / 1024 / 1024 and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})'
    }

    # 用于存储最终结果的字典
    metrics_data = {}

    for metric_name, promql in queries.items():
        try:
            results = prom.custom_query(query=promql)

            for item in results:
                instance = item["metric"].get("instance", "unknown")
                value = round(float(item["value"][1]), 2)  # 保留两位小数

                # 如果该 IP 还没在字典里，先初始化
                if instance not in metrics_data:
                    metrics_data[instance] = {
                        "last_update": datetime.now().strftime('%Y-%m-%d %H:%M:%S')
                    }

                # 将指标存入对应 IP
                metrics_data[instance][metric_name] = value

        except Exception as e:
            print(f"Query error for {metric_name}: {e}")

    # 将 Python 字典转换为 JSON 字符串
    return json.dumps(metrics_data, indent=4, ensure_ascii=False)


if __name__ == "__main__":
    # 调用函数
    json_result = get_node_metrics_json()

    # 打印结果
    print(json_result)
```

---

## 完整查询示例

```python
from prometheus_api_client import PrometheusConnect

# 初始化
prom = PrometheusConnect(url="http://10.17.151.170:9090", disable_ssl=True)

# 最近 10 分钟活跃的节点过滤条件
ACTIVE_NODES = 'rate(node_cpu_seconds_total{job="node"}[5m]) > 0'

# 查询最近活跃的节点数量
active_nodes = prom.custom_query(
    f'count by (instance) ({ACTIVE_NODES})'
)
print(f"最近活跃节点数量: {len(active_nodes)}")

# 查询活跃的 GPU 节点
active_gpu_nodes = prom.custom_query(
    f'count by (instance) (node_push_exporter_gpu_devices_detected{{job="node"}} > 0 and {ACTIVE_NODES})'
)
print(f"活跃 GPU 节点数量: {len(active_gpu_nodes)}")

# 过滤并计算 CPU 使用率（避免负数）
def get_cpu_usage(prom):
    """获取 CPU 使用率，仅返回有效节点"""
    query = f'''
    topk(10,
      clamp_min(
        (1 - avg by (instance) (rate(node_cpu_seconds_total{{job="node", mode="idle"}}[5m]))) * 100,
        0
      )
    )
    '''
    results = prom.custom_query(query)
    # 过滤有效结果
    return [m for m in results if float(m["value"][1]) >= 0]

# 过滤并计算内存使用率（避免负数）
def get_memory_usage(prom):
    """获取内存使用率，仅返回有效节点"""
    query = f'''
    topk(10,
      clamp_min(
        (1 - (node_memory_MemAvailable_bytes{{job="node"}} / node_memory_MemTotal_bytes{{job="node"}})) * 100,
        0
      )
    )
    '''
    results = prom.custom_query(query)
    # 过滤有效结果
    return [m for m in results if float(m["value"][1]) >= 0]

# 输出 CPU 使用率最高的机器
top_cpu = get_cpu_usage(prom)
print("\n=== CPU 使用率最高的 10 台机器 ===")
for m in top_cpu:
    instance = m["metric"]["instance"]
    cpu = float(m["value"][1])
    print(f"  {instance}: {cpu:.1f}%")

# 输出内存使用率最高的机器
top_mem = get_memory_usage(prom)
print("\n=== 内存使用率最高的 10 台机器 ===")
for m in top_mem:
    instance = m["metric"]["instance"]
    mem = float(m["value"][1])
    print(f"  {instance}: {mem:.1f}%")

# GPU 活跃节点状态
print("\n=== GPU 节点状态 ===")
gpu_query = f'''
clamp_min(gpu_utilization_percent{{job="node"}}, 0)
and on(instance) {ACTIVE_NODES}
'''
gpu_data = prom.custom_query(gpu_query)
for m in gpu_data:
    instance = m["metric"]["instance"]
    gpu = m["metric"]["gpu"]
    util = float(m["value"][1])
    print(f"  {instance} GPU {gpu}: {util:.1f}%")
```

### 关键函数说明

| 函数 | 作用 |
|------|------|
| `clamp_min(value, 0)` | 将负数结果限制为 0 |
| `rate(...[5m]) > 0` | 过滤最近活跃的节点 |
| `and on(instance)` | 按实例关联两个查询 |