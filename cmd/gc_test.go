package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aviadshiber/lightctl/internal/config"
)

func TestStateFileReadWrite(t *testing.T) {
	t.Parallel()
	cfg := config.NewConfig(nil, true)
	cfg.SetConfigDir(t.TempDir())

	// Write
	ids := []string{"act-1", "act-2", "act-3"}
	if err := cfg.WriteStateFile(ids); err != nil {
		t.Fatalf("WriteStateFile: %v", err)
	}

	// Read
	got, err := cfg.ReadStateFile()
	if err != nil {
		t.Fatalf("ReadStateFile: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 ids, got %d", len(got))
	}
	for i, id := range ids {
		if got[i] != id {
			t.Errorf("got[%d] = %q, want %q", i, got[i], id)
		}
	}
}

func TestStateFileEmpty(t *testing.T) {
	t.Parallel()
	cfg := config.NewConfig(nil, true)
	cfg.SetConfigDir(t.TempDir())

	got, err := cfg.ReadStateFile()
	if err != nil {
		t.Fatalf("ReadStateFile: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestStateFileAddRemove(t *testing.T) {
	t.Parallel()
	cfg := config.NewConfig(nil, true)
	cfg.SetConfigDir(t.TempDir())

	if err := cfg.AddActionToState("a"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.AddActionToState("b"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.AddActionToState("c"); err != nil {
		t.Fatal(err)
	}

	if err := cfg.RemoveActionFromState("b"); err != nil {
		t.Fatal(err)
	}

	got, _ := cfg.ReadStateFile()
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0] != "a" || got[1] != "c" {
		t.Fatalf("expected [a c], got %v", got)
	}
}

func TestStateFileFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := config.NewConfig(nil, true)
	cfg.SetConfigDir(dir)

	if err := cfg.WriteStateFile([]string{"x", "y"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "active-actions.json"))
	if err != nil {
		t.Fatal(err)
	}

	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("state file is not valid JSON array: %v", err)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr))
	}
}

func TestParseFileLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		wantFile string
		wantLine int
		wantErr  bool
	}{
		{"Foo.java:42", "Foo.java", 42, false},
		{"src/main/Bar.java:100", "src/main/Bar.java", 100, false},
		{"nocolon", "", 0, true},
		{"Foo.java:abc", "", 0, true},
		{"Foo.java:0", "", 0, true},
		{"Foo.java:-1", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			f, l, err := parseFileLine(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f != tt.wantFile || l != tt.wantLine {
				t.Fatalf("got (%q, %d), want (%q, %d)", f, l, tt.wantFile, tt.wantLine)
			}
		})
	}
}
