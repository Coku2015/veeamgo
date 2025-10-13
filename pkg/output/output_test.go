package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintTableSlice(t *testing.T) {
	type row struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}

	rows := []row{
		{Name: "alpha", ID: "1"},
		{Name: "beta", ID: "2"},
	}

	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	if err := printTable(rows); err != nil {
		t.Fatalf("printTable returned error: %v", err)
	}

	_ = w.Close()
	<-done
	os.Stdout = stdout

	out := buf.String()
	if !strings.Contains(out, "name") || !strings.Contains(out, "alpha") {
		t.Fatalf("unexpected table output: %q", out)
	}
}

func TestPrintTableNil(t *testing.T) {
	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	if err := printTable(nil); err != nil {
		t.Fatalf("printTable returned error: %v", err)
	}

	_ = w.Close()
	<-done
	os.Stdout = stdout

	out := buf.String()
	if !strings.Contains(out, "No records found.") {
		t.Fatalf("expected empty message, got %q", out)
	}
}
