package metrics

import "testing"

func TestMetricLine_SortsAndEscapesLabels(t *testing.T) {
	got := MetricLine("node_test_metric", map[string]string{
		"z": "line\nbreak",
		"a": `quote"slash\`,
	}, 1)

	want := "node_test_metric{a=\"quote\\\"slash\\\\\",z=\"line\\nbreak\"} 1"
	if got != want {
		t.Fatalf("MetricLine() = %q, want %q", got, want)
	}
}

func TestSanitizeComment(t *testing.T) {
	got := SanitizeComment("line1\nline2")
	if got != "line1 line2" {
		t.Fatalf("SanitizeComment() = %q, want %q", got, "line1 line2")
	}
}
