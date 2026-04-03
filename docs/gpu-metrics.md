# GPU 指标采集说明

本文档说明 `node-push-exporter` 当前已经实现的 GPU 指标采集行为，包括采集方式、失败处理、指标名称和标签。

## 工作方式

`node-push-exporter` 当前的推送链路是：

```text
node_exporter /metrics
  + GPU 采集结果
  -> 合并成一份 Prometheus 文本
  -> 推送到 Pushgateway
```

GPU 采集属于附加链路，不是主链路。

这意味着：

- 有 GPU 且采集成功时，GPU 指标会和 `node_exporter` 指标一起推送
- 没有 GPU 时，不会报错退出
- GPU 采集失败时，不影响 `node_exporter` 指标推送

## 当前支持的采集命令

程序会自动探测本机是否存在以下命令：

- `nvidia-smi`
- `rocm-smi`

如果命令存在且执行成功，就会采集对应厂商的 GPU 指标。

当前实现使用的命令为：

### NVIDIA

```bash
nvidia-smi -L
nvidia-smi --query-gpu=index,name,uuid,temperature.gpu,utilization.gpu,memory.total,memory.used,power.draw --format=csv,noheader,nounits
```

### ROCm / 海光

```bash
rocm-smi --showtemp --showuse --showmemuse --json
```

## 失败处理

GPU 采集失败不会让整次推送失败。

当前行为如下：

- `node_exporter` 抓取失败：本次推送失败
- Pushgateway 推送失败：本次推送失败
- GPU 采集失败：继续推送 `node_exporter` 指标
- 主机没有 GPU：继续推送 `node_exporter` 指标

GPU 采集失败时，日志中会出现类似信息：

```text
GPU指标采集失败，继续推送node_exporter指标: ...
```

## 标签说明

GPU 指标可能包含以下标签：

- `vendor`
  取值当前为 `nvidia` 或 `rocm`
- `gpu`
  逻辑编号，例如 `0`
- `name`
  设备型号，例如 `NVIDIA A800`
- `uuid`
  设备唯一标识，当前 NVIDIA 可提供
- `device_id`
  原始设备名，当前 ROCm 常见为 `card0`、`card1`

不同指标是否带哪些标签，取决于当前厂商命令能返回哪些字段。

## 指标列表

以下是当前代码实际会输出的 GPU 相关指标。

### 始终可能出现的状态指标

这些指标是 GPU 采集模块自己的状态指标。

#### `node_push_exporter_gpu_scrape_timestamp_seconds`

含义：
本次 GPU 采集时间戳，单位为 Unix seconds。

示例：

```text
node_push_exporter_gpu_scrape_timestamp_seconds 1712131200
```

说明：

- 只要程序进入 GPU 采集流程，就会输出
- 即使当前主机没有 GPU，也会输出这个指标

#### `node_push_exporter_gpu_scrape_success`

含义：
某个厂商的 GPU 采集是否成功。

标签：

- `vendor`

取值：

- `1` 表示该厂商采集成功
- `0` 表示该厂商命令存在但采集失败

示例：

```text
node_push_exporter_gpu_scrape_success{vendor="nvidia"} 1
node_push_exporter_gpu_scrape_success{vendor="rocm"} 0
```

说明：

- 如果某厂商命令不存在，则不会输出该厂商的这条指标
- 如果主机没有任何 GPU 命令，只会有时间戳指标，不会有 `scrape_success`

#### `node_push_exporter_gpu_devices_detected`

含义：
某个厂商本次识别到的 GPU 数量。

标签：

- `vendor`

示例：

```text
node_push_exporter_gpu_devices_detected{vendor="nvidia"} 4
node_push_exporter_gpu_devices_detected{vendor="rocm"} 0
```

### 设备级通用指标

这些指标在某张 GPU 的对应字段可获取时输出。

#### `gpu_up`

含义：
该设备在本次采集中可用。

常见标签：

- `vendor`
- `gpu`
- `name`
- `uuid`
- `device_id`

示例：

```text
gpu_up{vendor="nvidia",gpu="0",name="NVIDIA A800",uuid="GPU-xxx"} 1
```

#### `gpu_info`

含义：
设备信息指标，数值恒为 `1`，设备属性放在标签中。

常见标签：

- `vendor`
- `gpu`
- `name`
- `uuid`
- `device_id`

示例：

```text
gpu_info{vendor="nvidia",gpu="0",name="NVIDIA A800",uuid="GPU-xxx"} 1
```

#### `gpu_temperature_celsius`

含义：
设备温度，单位摄氏度。

当前来源：

- NVIDIA: `temperature.gpu`
- ROCm: 优先取 edge 温度

示例：

```text
gpu_temperature_celsius{vendor="nvidia",gpu="0",uuid="GPU-xxx"} 52
```

#### `gpu_utilization_percent`

含义：
设备利用率，单位百分比。

当前来源：

- NVIDIA: `utilization.gpu`
- ROCm: `GPU use (%)` 或 `HCU use (%)`

示例：

```text
gpu_utilization_percent{vendor="nvidia",gpu="0",uuid="GPU-xxx"} 78
```

#### `gpu_memory_used_percent`

含义：
显存使用率，单位百分比。

当前来源：

- ROCm: `GPU memory use (%)`、`GPU Memory Allocated (%)`、`VRAM use (%)` 或 `HCU memory use (%)`

说明：

- 当前 NVIDIA 实现不输出这条百分比指标
- ROCm 能拿到百分比时才会输出

#### `gpu_memory_used_bytes`

含义：
已用显存，单位 bytes。

当前来源：

- NVIDIA: `memory.used`

说明：

- 当前 ROCm 实现如果拿不到字节总量和已用量，则不会输出该指标

示例：

```text
gpu_memory_used_bytes{vendor="nvidia",gpu="0",uuid="GPU-xxx"} 42949672960
```

#### `gpu_memory_total_bytes`

含义：
显存总量，单位 bytes。

当前来源：

- NVIDIA: `memory.total`

说明：

- 当前 ROCm 实现默认不输出该指标，除非后续补充更完整字段

示例：

```text
gpu_memory_total_bytes{vendor="nvidia",gpu="0",uuid="GPU-xxx"} 85899345920
```

#### `gpu_power_draw_watts`

含义：
当前功耗，单位 watts。

当前来源：

- NVIDIA: `power.draw`

说明：

- 当前 ROCm 实现默认不输出该指标

示例：

```text
gpu_power_draw_watts{vendor="nvidia",gpu="0",uuid="GPU-xxx"} 250.5
```

### ROCm 温度细分指标

这些指标仅在 `rocm-smi --json` 返回对应字段时输出。

#### `gpu_temperature_edge_celsius`

含义：
edge 温度，单位摄氏度。

#### `gpu_temperature_junction_celsius`

含义：
junction 温度，单位摄氏度。

#### `gpu_temperature_mem_celsius`

含义：
显存温度，单位摄氏度。

#### `gpu_temperature_core_celsius`

含义：
core 温度，单位摄氏度。

示例：

```text
gpu_temperature_edge_celsius{vendor="rocm",gpu="0",device_id="card0"} 67
gpu_temperature_junction_celsius{vendor="rocm",gpu="0",device_id="card0"} 75
gpu_temperature_mem_celsius{vendor="rocm",gpu="0",device_id="card0"} 70
gpu_temperature_core_celsius{vendor="rocm",gpu="0",device_id="card0"} 68
```

## 指标出现规则

为了避免误解，当前实现遵循下面的规则：

- 指标只在字段可获取时输出
- 不会为了“看起来统一”而伪造缺失字段
- 没有 GPU 时，不输出 `gpu_*` 设备指标
- 某厂商命令不存在时，不输出该厂商的 `scrape_success` 和 `devices_detected`
- 某厂商命令存在但执行失败时，会输出该厂商的失败状态指标

## 当前实现边界

当前版本已经实现的范围：

- NVIDIA 基础温度、利用率、显存字节数、功耗
- ROCm 基础温度、利用率、显存使用百分比
- GPU 状态指标
- GPU 失败不影响 node 指标推送

当前还没有实现的内容：

- GPU 配置开关
- 自定义 GPU 采集超时配置
- NVIDIA MIG 指标
- NVIDIA XID 错误指标
- ROCm 显存 bytes 总量与已用量的完整支持

## 验证方式

可以通过以下方式确认 GPU 指标是否被推送：

### 1. 查看本地日志

```bash
journalctl -u node-push-exporter -f
```

### 2. 检查 Pushgateway

```bash
curl -s http://<pushgateway>/metrics | grep -E "gpu_|node_push_exporter_gpu_"
```

### 3. 关注无 GPU 主机行为

无 GPU 主机上属于正常现象：

- 没有 `gpu_*` 设备指标
- 可能只有 `node_push_exporter_gpu_scrape_timestamp_seconds`
- `node_exporter` 指标仍然持续推送
