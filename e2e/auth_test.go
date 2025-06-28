//go:build e2e

package e2e

import (
	"testing"
)

func TestAuthOperations(t *testing.T) {
	helper := NewE2ETestHelper(t)
	helper.LogTestInfo(t)

	t.Run("GetMe", func(t *testing.T) {
		user, err := helper.App.SDK.GetMe()
		if err != nil {
			t.Fatalf("Failed to get user info: %v", err)
		}

		if user.UserPrincipalName == "" {
			t.Error("Expected UserPrincipalName to be populated, but it was empty")
		}

		if user.ID == "" {
			t.Error("Expected user ID to be populated, but it was empty")
		}

		t.Logf("Successfully fetched user info for: %s", user.UserPrincipalName)
	})
}
