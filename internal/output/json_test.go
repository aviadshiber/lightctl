package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	data := map[string]string{"name": "test"}
	if err := PrintJSON(&buf, data); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	got := strings.TrimSpace(buf.String())
	if got != `{"name":"test"}` {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestPrintPrettyJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	data := map[string]string{"name": "test"}
	if err := PrintPrettyJSON(&buf, data); err != nil {
		t.Fatalf("PrintPrettyJSON: %v", err)
	}
	if !strings.Contains(buf.String(), "\n") {
		t.Fatal("expected multiline output")
	}
}

func TestFilterJQ(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data interface{}
		expr string
		want string
	}{
		{
			name: "identity",
			data: map[string]interface{}{"a": 1.0},
			expr: ".",
			want: `{"a":1}`,
		},
		{
			name: "field select",
			data: map[string]interface{}{"a": 1.0, "b": 2.0},
			expr: ".a",
			want: "1",
		},
		{
			name: "array index",
			data: []interface{}{10.0, 20.0, 30.0},
			expr: ".[1]",
			want: "20",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FilterJQ(tt.data, tt.expr)
			if err != nil {
				t.Fatalf("FilterJQ: %v", err)
			}
			got := strings.TrimSpace(formatResult(result))
			if got != tt.want {
				t.Errorf("FilterJQ(%q) = %q, want %q", tt.expr, got, tt.want)
			}
		})
	}
}

func TestFilterJQ_InvalidExpr(t *testing.T) {
	t.Parallel()
	_, err := FilterJQ(map[string]interface{}{}, ".[invalid")
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
}

func formatResult(v interface{}) string {
	var buf bytes.Buffer
	PrintJSON(&buf, v)
	return strings.TrimSpace(buf.String())
}
