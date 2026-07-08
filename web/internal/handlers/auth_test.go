package handlers

import (
	"testing"
)

func TestTemplateFuncs(t *testing.T) {
	funcs := templateFuncs()
	
	deref, ok := funcs["deref"].(func(*string) string)
	if !ok {
		t.Fatalf("deref function not found or wrong signature")
	}
	
	var str = "test"
	if deref(&str) != "test" {
		t.Errorf("Expected 'test', got %s", deref(&str))
	}
	if deref(nil) != "" {
		t.Errorf("Expected empty string for nil")
	}

	derefInt64, ok := funcs["derefInt64"].(func(*int64) int64)
	if !ok {
		t.Fatalf("derefInt64 function not found or wrong signature")
	}

	var i int64 = 42
	if derefInt64(&i) != 42 {
		t.Errorf("Expected 42, got %d", derefInt64(&i))
	}
	if derefInt64(nil) != 0 {
		t.Errorf("Expected 0 for nil")
	}
	
	formatBps, ok := funcs["formatBps"].(func(float64) string)
	if !ok {
		t.Fatalf("formatBps function not found or wrong signature")
	}
	
	if formatBps(1000) != "1.00 Kbps" {
		t.Errorf("Expected 1.00 Kbps, got %s", formatBps(1000))
	}
}
