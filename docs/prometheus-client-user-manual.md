# `prometheus-api-client` 使用手册


Prometheus 可以理解为服务器指标数据中心。它作为底层数据提供者，持续采集并存储服务器的运行状态数据，例如 CPU、内存、磁盘、网络等指标。上层的巡检、报表、告警、容量分析或自动化服务，都可以通过 Prometheus 提供的 HTTP API 和 PromQL 查询语言获取这些指标数据，再进一步加工成面向业务的能力。

* 常见查询对象包括 CPU、内存、磁盘、网络等


注意：前提条件是，目标机器上已安装node-push-exporter服务。
## 1. 安装

```bash
pip install prometheus-api-client
```

## 2. 使用前准备

开始写脚本前，通常需要确认下面几项：

* Prometheus 地址，例如 `http://10.17.154.252:9090`
* 目标机器的 `instance` 标签，例如 `10.17.51.24`
* 对应的 `job` 标签，例如 `node-exporter`




先确认目标实例是否存在，再继续写查询。

## 3. 连接 Prometheus

### 3.1 最小示例

```python
from prometheus_api_client import PrometheusConnect

prom = PrometheusConnect(
    url="http://10.17.154.252:9090",
    disable_ssl=True,
)

result = prom.custom_query(query="up")
print(result)
```

## 4. 常用查询方式

### 4.1 即时查询

查询某台机器：

```python
result = prom.custom_query(query='up{instance="10.17.51.24"}')
print(result)
```

说明：

* `custom_query()` 适合查当前值

## 5. 常用查询示例

下面的例子默认 Prometheus 已抓取 `node_exporter` 指标。

### 5.1 CPU 使用率

查询单台机器 CPU 总使用率：

Python 示例：

```python
instance = "10.17.51.24"
query = f'''
100 * (1 - avg by(instance) (
  rate(node_cpu_seconds_total{{mode="idle", instance="{instance}"}}[5m])
))
'''

result = prom.custom_query(query=query)
print(result)
```

查询每个 CPU 核心使用率：

```python
query = '''
100 * (1 - avg by(instance, cpu) (
  rate(node_cpu_seconds_total{mode="idle", instance="10.17.51.24"}[5m])
))
'''

result = prom.custom_query(query=query)
print(result)
```

### 5.2 内存使用率

查询内存使用率：

Python 示例：

```python
query = '''
100 * (
  1 - (
    node_memory_MemAvailable_bytes{instance="10.17.51.24"}
    /
    node_memory_MemTotal_bytes{instance="10.17.51.24"}
  )
)
'''

result = prom.custom_query(query=query)
print(result)
```

### 5.3 磁盘读写速率

查询磁盘读取速率：

```python
query = '''
sum by(instance) (
  rate(node_disk_read_bytes_total{instance="10.17.51.24"}[5m])
)
'''

result = prom.custom_query(query=query)
print(result)
```

查询磁盘写入速率：

```python
query = '''
sum by(instance) (
  rate(node_disk_written_bytes_total{instance="10.17.51.24"}[5m])
)
'''

result = prom.custom_query(query=query)
print(result)
```

### 5.4 网络流量

常见网络查询通常只看两类：

* 单台机器总入流量 / 总出流量
* 按网卡查看流量

下面的查询都排除了回环网卡 `lo`。

查询单台机器总入流量：

```python
query = '''
sum by(instance) (
  rate(node_network_receive_bytes_total{instance="10.17.51.24", device!~"lo"}[5m])
)
'''

result = prom.custom_query(query=query)
print(result)
```

查询单台机器总出流量：

```python
query = '''
sum by(instance) (
  rate(node_network_transmit_bytes_total{instance="10.17.51.24", device!~"lo"}[5m])
)
'''

result = prom.custom_query(query=query)
print(result)
```


## 6. 建议的脚本模板

下面是一个适合直接改造的最小脚本：

```python
from prometheus_api_client import PrometheusConnect

PROM_URL = "http://10.17.154.252:9090"
INSTANCE = "10.17.51.24"

prom = PrometheusConnect(url=PROM_URL, disable_ssl=True)

query = f'''
100 * (1 - avg by(instance) (
  rate(node_cpu_seconds_total{{mode="idle", instance="{INSTANCE}"}}[5m])
))
'''

result = prom.custom_query(query=query)

for item in result:
    print(item["metric"])
    print(item["value"])
```

## 7. 常见问题

### 7.1 不知道 `instance` 是什么

先查：

```python
result = prom.custom_query(query="up")
print(result)
```

返回结果里的 `metric.instance` 就是你可以直接拿来过滤的值。

### 7.2 为什么很多查询都要用 `rate()`

因为像 CPU、磁盘、网络这类指标很多是累计值，直接看原始值意义不大，通常要先转成“每秒变化速率”。

### 7.3 查询结果为空

建议按这个顺序排查：

* 先查 `up`，确认 Prometheus 本身能返回数据
* 再去掉 `instance`、`job` 等过滤条件试一次
* 确认指标名称是否正确
* 确认目标 exporter 是否正在暴露该指标
