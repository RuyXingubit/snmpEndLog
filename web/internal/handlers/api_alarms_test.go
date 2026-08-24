package handlers

import (
	"strings"
	"testing"
	"time"
)

func TestAlarmStruct(t *testing.T) {
	now := time.Now()
	resolved := now.Add(5 * time.Minute)
	alarm := Alarm{
		ID:         1,
		DeviceID:   10,
		DeviceName: "Router-Core",
		EntityType: "interface",
		EntityID:   "ge-0/0/1",
		Name:       "Interface ge-0/0/1 Down",
		Severity:   "critical",
		Status:     "resolved",
		Message:    "Link down on port ge-0/0/1",
		CreatedAt:  now,
		ResolvedAt: &resolved,
	}

	if alarm.ID != 1 {
		t.Errorf("Expected ID 1, got %d", alarm.ID)
	}
	if alarm.DeviceID != 10 {
		t.Errorf("Expected DeviceID 10, got %d", alarm.DeviceID)
	}
	if alarm.DeviceName != "Router-Core" {
		t.Errorf("Expected DeviceName Router-Core, got %s", alarm.DeviceName)
	}
	if alarm.EntityType != "interface" {
		t.Errorf("Expected EntityType interface, got %s", alarm.EntityType)
	}
	if alarm.EntityID != "ge-0/0/1" {
		t.Errorf("Expected EntityID ge-0/0/1, got %s", alarm.EntityID)
	}
	if alarm.Name != "Interface ge-0/0/1 Down" {
		t.Errorf("Expected Name 'Interface ge-0/0/1 Down', got %s", alarm.Name)
	}
	if alarm.Severity != "critical" {
		t.Errorf("Expected Severity critical, got %s", alarm.Severity)
	}
	if alarm.Status != "resolved" {
		t.Errorf("Expected Status resolved, got %s", alarm.Status)
	}
	if alarm.Message != "Link down on port ge-0/0/1" {
		t.Errorf("Expected Message 'Link down on port ge-0/0/1', got %s", alarm.Message)
	}
	if !alarm.CreatedAt.Equal(now) {
		t.Errorf("Expected CreatedAt to be %v, got %v", now, alarm.CreatedAt)
	}
	if alarm.ResolvedAt == nil || *alarm.ResolvedAt != resolved {
		t.Errorf("Expected ResolvedAt to be set correctly")
	}
}

func TestBuildAlarmsQuery(t *testing.T) {
	tests := []struct {
		name           string
		statusFilter   string
		deviceIDFilter string
		expectWhere    bool
		expectStatus   bool
		expectDevice   bool
		expectedArgs   int
	}{
		{
			name:           "Default active status with no device",
			statusFilter:   "active",
			deviceIDFilter: "",
			expectWhere:    true,
			expectStatus:   true,
			expectDevice:   false,
			expectedArgs:   1,
		},
		{
			name:           "All status with no device",
			statusFilter:   "all",
			deviceIDFilter: "",
			expectWhere:    false,
			expectStatus:   false,
			expectDevice:   false,
			expectedArgs:   0,
		},
		{
			name:           "Active status with device filter",
			statusFilter:   "active",
			deviceIDFilter: "5",
			expectWhere:    true,
			expectStatus:   true,
			expectDevice:   true,
			expectedArgs:   2,
		},
		{
			name:           "Resolved status with device filter",
			statusFilter:   "resolved",
			deviceIDFilter: "12",
			expectWhere:    true,
			expectStatus:   true,
			expectDevice:   true,
			expectedArgs:   2,
		},
		{
			name:           "All status with device filter",
			statusFilter:   "all",
			deviceIDFilter: "8",
			expectWhere:    true,
			expectStatus:   false,
			expectDevice:   true,
			expectedArgs:   1,
		},
		{
			name:           "Invalid device filter ignored",
			statusFilter:   "active",
			deviceIDFilter: "invalid",
			expectWhere:    true,
			expectStatus:   true,
			expectDevice:   false,
			expectedArgs:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args := buildAlarmsQuery(tt.statusFilter, tt.deviceIDFilter)

			if len(args) != tt.expectedArgs {
				t.Errorf("Expected %d args, got %d (%v)", tt.expectedArgs, len(args), args)
			}

			hasWhere := strings.Contains(query, "WHERE")
			if hasWhere != tt.expectWhere {
				t.Errorf("Expected WHERE clause %v, got %v in query: %s", tt.expectWhere, hasWhere, query)
			}

			hasStatus := strings.Contains(query, "a.status =")
			if hasStatus != tt.expectStatus {
				t.Errorf("Expected status condition %v, got %v in query: %s", tt.expectStatus, hasStatus, query)
			}

			hasDevice := strings.Contains(query, "a.device_id = $")
			if hasDevice != tt.expectDevice {
				t.Errorf("Expected device condition %v, got %v in query: %s", tt.expectDevice, hasDevice, query)
			}

			if !strings.Contains(query, "ORDER BY a.created_at DESC") {
				t.Errorf("Expected ORDER BY clause in query: %s", query)
			}
		})
	}
}
