---
title: 安装部署
description: Node Push Exporter 完整安装指南
---

# 安装部署

## 前置条件

- Linux 操作系统
- 具有 sudo 权限
- 网络可访问安装服务器

## 一键安装

```bash
curl -fsSL http://10.17.154.252:8888/download/install.sh | sudo bash
```

## 安装流程

安装脚本自动完成以下步骤：

| 步骤 | 说明 |
|------|------|
| 1 | 检测系统架构 (amd64/arm64/armv7) |
| 2 | 下载 node_exporter 和 node-push-exporter |
| 3 | 安装到 `/usr/local/bin/` |
| 4 | 创建配置目录 `/etc/node-push-exporter` |
| 5 | 生成 `config.yaml` 配置文件 |
| 6 | 写入 systemd 服务文件 |
| 7 | 启动服务并设置开机自启 |
| 8 | 健康检查 |

## 安装输出示例

```
[INFO] 正在检测系统架构...
[INFO] 检测到: linux/amd64
[INFO] 正在下载 node_exporter...
[INFO] 正在下载 node-push-exporter...
[INFO] 正在创建配置目录...
[INFO] 正在写入配置文件...
[INFO] 正在安装 systemd 服务...
[INFO] 正在启动服务...
[SUCCESS] Node Push Exporter 安装完成！
● node-push-exporter.service - Node Push Exporter
     Loaded: loaded (/etc/systemd/system/node-push-exporter.service; enabled)
     Active: active (running)
```

## 文件位置

| 文件 | 路径 |
|------|------|
| 二进制 | `/usr/local/bin/node-push-exporter` |
| node_exporter | `/usr/local/bin/node_exporter` |
| 配置文件 | `/etc/node-push-exporter/config.yaml` |
| systemd 服务 | `/etc/systemd/system/node-push-exporter.service` |

## 配置说明

编辑配置文件：

```bash
sudo vim /etc/node-push-exporter/config.yaml
```

主要配置项：

```ini
# Pushgateway 设置
pushgateway.url=http://your-prometheus:9091
pushgateway.job=node
pushgateway.interval=60

# 控制面（可选，用于节点注册）
control_plane.url=http://10.17.154.252:8080
```

修改配置后重启服务：

```bash
sudo systemctl restart node-push-exporter
```

## 服务管理

```bash
# 查看状态
sudo systemctl status node-push-exporter

# 查看日志
sudo journalctl -u node-push-exporter -f

# 重启
sudo systemctl restart node-push-exporter

# 停止
sudo systemctl stop node-push-exporter
```

## 卸载

### 自动卸载

```bash
curl -fsSL http://10.17.154.252:8888/download/uninstall.sh | sudo bash
```

### 手动卸载

```bash
sudo systemctl stop node-push-exporter
sudo systemctl disable node-push-exporter
sudo rm /etc/systemd/system/node-push-exporter.service
sudo systemctl daemon-reload
sudo rm /usr/local/bin/node_exporter
sudo rm /usr/local/bin/node-push-exporter
sudo rm -rf /etc/node-push-exporter
```

## 常见问题

### 安装失败，提示权限不足

确保使用 `sudo`：

```bash
curl -fsSL http://10.17.154.252:8888/download/install.sh | sudo bash
```

### 服务启动成功但无法推送

1. 检查服务状态：`sudo systemctl status node-push-exporter`
2. 确认 Pushgateway 地址可达
3. 查看日志：`sudo journalctl -u node-push-exporter -e`

### 如何修改推送间隔

```bash
sudo vim /etc/node-push-exporter/config.yaml
```

修改 `pushgateway.interval` 后：

```bash
sudo systemctl restart node-push-exporter
```