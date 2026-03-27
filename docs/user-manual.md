# node-push-exporter 用户手册

本文档面向使用者，介绍 `node-push-exporter` 的安装、运行、卸载和常见检查方式。

## 这个项目做什么

这个项目用于在服务器上自动运行两个组件：

- `node_exporter`：采集本机 CPU、内存、磁盘、网络等指标
- `node-push-exporter`：抓取本机 `node_exporter` 的指标，并推送到 Prometheus Pushgateway

典型流程如下：

1. 启动本机 `node_exporter`
2. 暴露本机指标地址，默认是 `http://127.0.0.1:9100/metrics`
3. 启动 `node-push-exporter`
4. `node-push-exporter` 定时抓取上述指标
5. 将指标推送到 Pushgateway

## 适用环境

- Linux
- 使用 `systemd`
- 需要 `root` 或 `sudo` 权限

## 一键安装

执行：

```bash
curl -fsSL http://10.17.154.252:8888/download/install.sh | sudo bash
```

### 安装脚本做了什么

`install.sh` 会自动完成以下工作：

1. 根据当前机器架构下载对应版本的 `node_exporter` 和 `node-push-exporter`
2. 将二进制安装到：
   - `/usr/local/bin/node_exporter`
   - `/usr/local/bin/node-push-exporter`
3. 创建或保留配置目录：
   - `/etc/node-push-exporter`
4. 如果系统还没有配置文件，则写入：
   - `/etc/node-push-exporter/config.yaml`
5. 写入 systemd 服务文件：
   - `/etc/systemd/system/node_exporter.service`
   - `/etc/systemd/system/node-push-exporter.service`
6. 在支持 SELinux 的系统上自动执行 `restorecon`，修复文件安全上下文
7. 执行 `systemctl daemon-reload`
8. 启动并设置开机自启：
   - `node_exporter`
   - `node-push-exporter`
9. 启动后自动检查服务状态和健康情况

### 安装完成后会检查什么

安装脚本不会只看“命令是否执行成功”，还会继续检查：

- `node_exporter` 是否已经是 `active`
- `node_exporter` 的指标地址是否可访问
- `node-push-exporter` 是否已经是 `active`
- `node-push-exporter` 配置中的 `node_exporter.metrics_url` 是否可访问

如果检查失败，脚本会直接输出诊断信息，包括：

- `systemctl status`
- `journalctl -u ... -n 50`
- 失败的探测地址

## 一键卸载

执行：

```bash
curl -fsSL http://10.17.154.252:8888/download/uninstall.sh | sudo bash
```

### 卸载脚本做了什么

`uninstall.sh` 会尽量幂等地完成彻底卸载：

1. 停止并禁用 `node_exporter`
2. 停止并禁用 `node-push-exporter`
3. 删除 systemd 服务文件
4. 删除二进制文件：
   - `/usr/local/bin/node_exporter`
   - `/usr/local/bin/node-push-exporter`
5. 删除配置目录：
   - `/etc/node-push-exporter`
6. 执行 `systemctl daemon-reload`

如果某个服务、文件或目录已经不存在，脚本会自动跳过，不会因为“未找到”而报错中断。

## 常用检查命令

安装后建议执行以下命令确认状态：

```bash
systemctl status node_exporter --no-pager
systemctl status node-push-exporter --no-pager
curl http://127.0.0.1:9100/metrics
journalctl -u node_exporter -n 50 --no-pager
journalctl -u node-push-exporter -n 50 --no-pager
```

## 常见问题

### 1. 提示需要 root 权限

请使用 `sudo` 执行脚本，例如：

```bash
curl -fsSL http://10.17.154.252:8888/download/install.sh | sudo bash
```

### 2. `node_exporter` 启动失败

优先检查：

- 二进制是否存在：`ls -l /usr/local/bin/node_exporter`
- 服务状态：`systemctl status node_exporter --no-pager`
- 服务日志：`journalctl -u node_exporter -n 50 --no-pager`
- 指标地址是否可访问：`curl http://127.0.0.1:9100/metrics`

如果是 SELinux 环境，还可以执行：

```bash
restorecon -Rv /usr/local/bin/node_exporter /etc/systemd/system/node_exporter.service
systemctl daemon-reload
systemctl restart node_exporter
```

### 3. `node-push-exporter` 启动失败

优先检查：

- 配置文件是否存在：`ls -l /etc/node-push-exporter/config.yaml`
- 服务状态：`systemctl status node-push-exporter --no-pager`
- 服务日志：`journalctl -u node-push-exporter -n 50 --no-pager`

### 4. 卸载时报“某个服务不存在”

新版 `uninstall.sh` 已经做了兼容处理。即使服务已经不存在，也会直接跳过，不影响卸载继续执行。

## 用户最常用的两条命令

安装：

```bash
curl -fsSL http://10.17.154.252:8888/download/install.sh | sudo bash
```

卸载：

```bash
curl -fsSL http://10.17.154.252:8888/download/uninstall.sh | sudo bash
```
