package handlers

import (
	"testing"
	"time"
)

func TestUserEntryStruct(t *testing.T) {
	now := time.Now()
	u := UserEntry{
		ID:        1,
		Username:  "admin",
		Role:      "admin",
		CreatedAt: now,
	}

	if u.ID != 1 {
		t.Errorf("Expected ID 1, got %d", u.ID)
	}
	if u.Username != "admin" {
		t.Errorf("Expected Username 'admin', got %s", u.Username)
	}
	if u.Role != "admin" {
		t.Errorf("Expected Role 'admin', got %s", u.Role)
	}
	if u.CreatedAt != now {
		t.Errorf("Expected CreatedAt %v, got %v", now, u.CreatedAt)
	}
}

func TestValidateUserEdit(t *testing.T) {
	tests := []struct {
		name          string
		currentUserID int
		targetUserID  int
		role          string
		password      string
		expectError   bool
		expectedErr   string
	}{
		{
			name:          "Valid role change to viewer for another user",
			currentUserID: 1,
			targetUserID:  2,
			role:          "viewer",
			password:      "",
			expectError:   false,
		},
		{
			name:          "Valid role change to admin for another user",
			currentUserID: 1,
			targetUserID:  2,
			role:          "admin",
			password:      "",
			expectError:   false,
		},
		{
			name:          "Valid role and new password",
			currentUserID: 1,
			targetUserID:  2,
			role:          "viewer",
			password:      "newpassword123",
			expectError:   false,
		},
		{
			name:          "Admin editing self maintaining admin role",
			currentUserID: 1,
			targetUserID:  1,
			role:          "admin",
			password:      "",
			expectError:   false,
		},
		{
			name:          "Invalid user ID",
			currentUserID: 1,
			targetUserID:  0,
			role:          "admin",
			password:      "",
			expectError:   true,
			expectedErr:   "ID de usuário inválido.",
		},
		{
			name:          "Invalid negative user ID",
			currentUserID: 1,
			targetUserID:  -5,
			role:          "admin",
			password:      "",
			expectError:   true,
			expectedErr:   "ID de usuário inválido.",
		},
		{
			name:          "Invalid role",
			currentUserID: 1,
			targetUserID:  2,
			role:          "superuser",
			password:      "",
			expectError:   true,
			expectedErr:   "Role inválida.",
		},
		{
			name:          "Empty role",
			currentUserID: 1,
			targetUserID:  2,
			role:          "",
			password:      "",
			expectError:   true,
			expectedErr:   "Role inválida.",
		},
		{
			name:          "Self demotion prevention (admin changing own role to viewer)",
			currentUserID: 1,
			targetUserID:  1,
			role:          "viewer",
			password:      "",
			expectError:   true,
			expectedErr:   "Você não pode alterar seu próprio perfil para viewer.",
		},
		{
			name:          "Short password (less than 8 chars)",
			currentUserID: 1,
			targetUserID:  2,
			role:          "viewer",
			password:      "1234567",
			expectError:   true,
			expectedErr:   "A senha deve ter no mínimo 8 caracteres.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserEdit(tt.currentUserID, tt.targetUserID, tt.role, tt.password)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedErr)
				} else if err.Error() != tt.expectedErr {
					t.Errorf("Expected error %q, got %q", tt.expectedErr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
