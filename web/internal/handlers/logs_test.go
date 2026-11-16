package handlers

import (
	"strings"
	"testing"
	"time"
)

func TestBuildLogsFilterWhere_RegisteredDeviceAndPlaceholders(t *testing.T) {
	t.Run("default filter requires registered enabled device", func(t *testing.T) {
		tf := TimeFilter{Period: "1 hour"}
		where, args, nextIdx := buildLogsFilterWhere(tf, "", "", "", false)

		if !strings.Contains(where, registeredDeviceLogsWhere) {
			t.Fatalf("expected registeredDeviceLogsWhere in %q", where)
		}
		if !strings.Contains(where, "time > NOW() - $1::interval") {
			t.Fatalf("expected time interval placeholder $1 in %q", where)
		}
		if len(args) != 1 || args[0] != "1 hour" {
			t.Fatalf("unexpected args: %#v", args)
		}
		if nextIdx != 2 {
			t.Fatalf("nextIdx = %d, want 2", nextIdx)
		}
	})

	t.Run("host severity and fulltext search sequential placeholders", func(t *testing.T) {
		tf := TimeFilter{Period: "24 hours"}
		where, args, nextIdx := buildLogsFilterWhere(tf, "BNG-PROSERV", "error", "bgp down", false)

		if !strings.Contains(where, registeredDeviceLogsWhere) {
			t.Fatalf("expected registeredDeviceLogsWhere in %q", where)
		}
		expected := []string{
			"time > NOW() - $1::interval",
			"host = $2",
			"severity_name = $3",
			"(to_tsvector('simple', message) @@ plainto_tsquery('simple', $4) OR app_name ILIKE $5)",
		}
		for _, fragment := range expected {
			if !strings.Contains(where, fragment) {
				t.Errorf("missing %q in where clause %q", fragment, where)
			}
		}
		if len(args) != 5 {
			t.Fatalf("len(args) = %d, want 5 (%#v)", len(args), args)
		}
		if args[1] != "BNG-PROSERV" || args[2] != "error" || args[3] != "bgp down" || args[4] != "%bgp down%" {
			t.Fatalf("unexpected args: %#v", args)
		}
		if nextIdx != 6 {
			t.Fatalf("nextIdx = %d, want 6", nextIdx)
		}
	})

	t.Run("custom time range host severity and exact ILIKE placeholders", func(t *testing.T) {
		start := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
		end := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
		tf := TimeFilter{IsCustom: true, Start: start, End: end}
		where, args, nextIdx := buildLogsFilterWhere(tf, "SW-CORE-S6730", "warning", "PKT_OUTQUEDROP", true)

		if !strings.Contains(where, registeredDeviceLogsWhere) {
			t.Fatalf("expected registeredDeviceLogsWhere in %q", where)
		}
		expected := []string{
			"time >= $1 AND time <= $2",
			"host = $3",
			"severity_name = $4",
			"(message ILIKE $5 OR app_name ILIKE $5)",
		}
		for _, fragment := range expected {
			if !strings.Contains(where, fragment) {
				t.Errorf("missing %q in where clause %q", fragment, where)
			}
		}
		if len(args) != 5 {
			t.Fatalf("len(args) = %d, want 5 (%#v)", len(args), args)
		}
		if args[0] != start || args[1] != end || args[2] != "SW-CORE-S6730" || args[3] != "warning" || args[4] != "%PKT_OUTQUEDROP%" {
			t.Fatalf("unexpected args: %#v", args)
		}
		if nextIdx != 6 {
			t.Fatalf("nextIdx = %d, want 6", nextIdx)
		}
	})
}
