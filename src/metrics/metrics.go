package metrics

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// MetricLine 渲染 Prometheus 指标行，对标签按键排序以保证输出稳定。
func MetricLine(name string, labels map[string]string, value any) string {
	if len(labels) == 0 {
		return fmt.Sprintf("%s %v", name, value)
	}

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, key, EscapeLabelValue(labels[key])))
	}

	return fmt.Sprintf("%s{%s} %v", name, strings.Join(parts, ","), value)
}

// MetricLineFloat 渲染 Prometheus 指标行，使用 float64 值以获得精确格式化。
func MetricLineFloat(name string, labels map[string]string, value float64) string {
	if len(labels) == 0 {
		return fmt.Sprintf("%s %s", name, strconv.FormatFloat(value, 'f', -1, 64))
	}

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, key, EscapeLabelValue(labels[key])))
	}

	return fmt.Sprintf("%s{%s} %s", name, strings.Join(parts, ","), strconv.FormatFloat(value, 'f', -1, 64))
}

// EscapeLabelValue 转义 Prometheus 标签值中的反斜杠、换行符和双引号。
func EscapeLabelValue(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, "\n", `\n`, `"`, `\"`)
	return replacer.Replace(value)
}

// SanitizeComment 将错误消息压缩为单行，以避免破坏指标文本格式。
func SanitizeComment(value string) string {
	return strings.ReplaceAll(value, "\n", " ")
}
