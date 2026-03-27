package pusher

/*
	Pushgateway推送器

	该模块负责将格式化后的指标数据推送到Prometheus Pushgateway。

	Pushgateway API说明:
	  - URL格式: http://pushgateway:9091/metrics/job/<job_name>[/instance/<instance_name>]
	  - HTTP方法: POST
	  - Content-Type: text/plain; version=0.0.4

	功能特性:
	  - 支持自定义job名称
	  - 支持instance标签
	  - 可配置HTTP超时
*/

import (
	"bytes"    // 字节缓冲区
	"fmt"      // 格式化错误
	"io"       // IO操作
	"net/http" // HTTP客户端
	"time"     // 超时设置
)

// Pusher Pushgateway推送客户端
type Pusher struct {
	url        string       // Pushgateway基础地址，例如 http://pushgateway:9091
	job        string       // 推送路径中的 job 标签
	instance   string       // 推送路径中的 instance 标签，可选
	httpClient *http.Client // 发送 HTTP 请求的客户端，可配置超时
}

// Option Pusher配置选项函数类型
// 用于通过函数式选项模式配置Pusher
type Option func(*Pusher)

// WithJob 设置任务名称
// 参数: job 任务名称，用于在Pushgateway中标识这批指标
func WithJob(job string) Option {
	return func(p *Pusher) {
		p.job = job
	}
}

// WithInstance 设置实例名称
// 参数: instance 实例标识，如主机名
func WithInstance(instance string) Option {
	return func(p *Pusher) {
		p.instance = instance
	}
}

// WithTimeout 设置HTTP请求超时时间
// 参数: timeout 超时时长
func WithTimeout(timeout time.Duration) Option {
	return func(p *Pusher) {
		p.httpClient.Timeout = timeout
	}
}

// NewPusher 创建Pushgateway推送客户端
// 参数:
//   - url: Pushgateway地址，如http://localhost:9091
//   - opts: 可变参数，用于配置job、instance、timeout等
func NewPusher(url string, opts ...Option) *Pusher {
	// 默认 job 为 node，和当前项目“推送 node_exporter 指标”的定位保持一致。
	p := &Pusher{
		url: url,
		job: "node", // 默认任务名
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // 默认超时10秒
		},
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(p)
	}

	return p
}

/*
Push 推送指标到Pushgateway

参数:
metrics - 格式化后的Prometheus指标文本

返回:
error - 推送过程中的错误信息

推送URL格式: http://pushgateway/metrics/job/<job>[/instance/<instance>]
*/
func (p *Pusher) Push(metrics []byte) error {
	// Pushgateway 通过 URL 路径携带 job/instance，而不是放在查询参数里。
	pushURL := p.url + "/metrics/job/" + p.job
	// 如果指定了instance，添加到URL中
	if p.instance != "" {
		pushURL += "/instance/" + p.instance
	}

	// 创建HTTP POST请求
	req, err := http.NewRequest("POST", pushURL, bytes.NewReader(metrics))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	// Pushgateway要求使用Prometheus文本格式版本0.0.4
	req.Header.Set("Content-Type", "text/plain; version=0.0.4")

	// Pushgateway 接收的是原始 Prometheus 文本内容，因此请求体直接写 metrics。
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("推送指标失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查HTTP响应状态码
	// 2xx表示成功，其他表示失败
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 读取响应体用于错误诊断
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("推送失败，状态码%d: %s", resp.StatusCode, string(body))
	}

	return nil
}
