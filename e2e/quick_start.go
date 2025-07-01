// Quick Start E2E Validation
// This file contains a simple validation test to check if E2E setup is working
package e2e

import (
	"context"
	"testing"
)

// TestE2ESetupValidation is a minimal test to validate E2E configuration
func TestE2ESetupValidation(t *testing.T) {
	// Test 1: Configuration Loading
	cfg := LoadConfig()

	t.Logf("✓ Configuration loaded successfully")
	t.Logf("  Test Directory: %s", cfg.TestDir)
	t.Logf("  Timeout: %v", cfg.Timeout)
	t.Logf("  Cleanup: %v", cfg.Cleanup)

	// Test 2: Authentication
	helper := NewE2ETestHelper(t)

	t.Logf("✓ Authentication successful")
	t.Logf("  Test ID: %s", helper.TestID)
	t.Logf("  Test Directory: %s", helper.TestDir)

	// Test 3: Basic Drive Access
	drives, err := helper.App.SDK.GetDrives(context.Background())
	if err != nil {
		t.Fatalf("Failed to list drives: %v", err)
	}

	if len(drives.Value) == 0 {
		t.Fatal("No drives found - check account setup")
	}

	t.Logf("✓ Drive access working")
	t.Logf("  Found %d drive(s)", len(drives.Value))
	for i, drive := range drives.Value {
		t.Logf("    Drive %d: %s (%s)", i+1, drive.Name, drive.DriveType)
	}

	// Test 4: Test Directory Creation
	testFile := "validation-test.txt"
	testContent := []byte("Hello from E2E validation test!")

	localFile := helper.CreateTestFile(t, testFile, testContent)
	remotePath := helper.GetTestPath(testFile)

	_, err = helper.App.SDK.UploadFile(context.Background(), localFile, remotePath)
	if err != nil {
		t.Fatalf("Failed to upload validation file: %v", err)
	}

	t.Logf("✓ File upload successful")
	t.Logf("  Local: %s", localFile)
	t.Logf("  Remote: %s", remotePath)

	// Test 5: File Verification
	helper.AssertFileExists(t, remotePath)
	helper.CompareFileContent(t, remotePath, testContent)

	t.Logf("✓ File verification successful")
	t.Logf("  Content verified: %d bytes", len(testContent))

	// Success message
	t.Log("")
	t.Log("🎉 E2E Setup Validation PASSED!")
	t.Log("Your E2E testing environment is ready to use.")
	t.Log("")
	t.Log("Next steps:")
	t.Log("  1. Run full test suite: go test -tags=e2e -v ./e2e/...")
	t.Log("  2. Check the e2e/README.md for more testing options")
	t.Log("  3. Monitor your test OneDrive account for the test data")
}

// maskSensitive masks sensitive information for logging
func maskSensitive(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
