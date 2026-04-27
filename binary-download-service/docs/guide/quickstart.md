---
title: 快速开始
description: 一键部署 Node Push Exporter，1 分钟完成
---

# 快速开始

## 前置条件

- Linux 操作系统
- sudo 权限
- 网络可访问安装服务器

## 一键安装

```bash
curl -fsSL http://10.17.154.252:8888/download/install.sh | sudo bash
```

安装过程会自动完成：下载二进制、创建配置、注册 systemd 服务、启动。

## 验证部署

### 检查服务状态

```bash
sudo systemctl status node-push-exporter
```

### 检查指标

```bash
curl http://localhost:9100/metrics | head -10
```

### 检查日志

```bash
sudo journalctl -u node-push-exporter -f --no-pager
```

## 下一步

- [配置 Prometheus 采集](/metrics/prometheus)
- [查看指标查询示例](/metrics/prometheus#可用指标清单)
- [完整安装文档](/guide/install)