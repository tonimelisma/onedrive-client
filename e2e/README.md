# End-to-End Testing for onedrive-client

This directory contains end-to-end (E2E) tests that run against a real OneDrive account to validate the complete functionality of the OneDrive client.

## Quick Start

**1. Log in with the CLI** (if you haven't already):
```bash
./onedrive-client auth login
./onedrive-client auth status  # Verify you're logged in
```

**2. Copy your config to the project root**:
```bash
cp ~/.config/onedrive-client/config.json ./config.json
```

**3. Run E2E tests**:
```bash
# Quick validation test
go test -tags=e2e -v ./e2e/ -run TestE2ESetupValidation

# Full test suite
go test -tags=e2e -v ./e2e/...
```

That's it! The E2E tests use the same authentication as the CLI.

## How It Works

- **Same Authentication**: E2E tests use your existing CLI login (same device code flow)
- **Token Reuse**: Uses your `config.json` with refresh tokens for seamless testing
- **Auto Refresh**: Tokens are automatically refreshed during testing
- **Isolated Testing**: All test data goes to `/E2E-Tests/` directory 
- **Safe**: Never touches your existing OneDrive data

## Test Structure

### Test Categories

**File Operations** (`files_test.go`):
- Directory creation and listing
- Small file upload/download (< 4MB)
- Large file session management (> 4MB)
- File metadata and content verification
- Upload session status and cancellation

**Drive Operations**:
- List all accessible drives
- Get default drive information
- Check quota and usage information

**Error Handling**:
- Non-existent file scenarios
- Invalid session management
- Network error recovery

## Configuration Options

Set these environment variables to customize test behavior:

```bash
# Optional settings
export ONEDRIVE_E2E_TEST_DIR="/E2E-Tests"     # Where to put test files
export ONEDRIVE_E2E_TIMEOUT="300s"            # Operation timeout  
export ONEDRIVE_E2E_CLEANUP="true"            # Auto-cleanup test data
export ONEDRIVE_E2E_MAX_FILE_SIZE="104857600" # 100MB max test file size
```

## Running Tests

### Different Test Scopes

```bash
# Quick validation (fastest)
go test -tags=e2e -v ./e2e/ -run TestE2ESetupValidation

# File operations only
go test -tags=e2e -v ./e2e/ -run TestFileOperations  

# Drive operations only
go test -tags=e2e -v ./e2e/ -run TestDriveOperations

# Error handling tests
go test -tags=e2e -v ./e2e/ -run TestErrorHandling

# Everything
go test -tags=e2e -v ./e2e/...

# With longer timeout for slow connections
go test -tags=e2e -v -timeout=10m ./e2e/...
```

### Debug Mode

```bash
# Keep test data for inspection
ONEDRIVE_E2E_CLEANUP=false go test -tags=e2e -v ./e2e/...

# Run single test for debugging
go test -tags=e2e -v ./e2e/ -run TestFileOperations/UploadSmallFile
```

## Safety Features

### Account Protection
- **Isolated Directory**: All tests use `/E2E-Tests/` directory only
- **No Data Pollution**: Never modifies your existing OneDrive files
- **Read-Only Personal Data**: Only reads drive info, never changes it
- **Automatic Cleanup**: Removes test files after completion (configurable)

### Data Management
- **Unique Test IDs**: Each test run gets a unique subdirectory (`e2e-{timestamp}`)
- **Parallel Safe**: Multiple test runs can execute simultaneously
- **Collision Avoidance**: Tests never interfere with each other
- **Cleanup Verification**: Logs cleanup status and any issues

## Test Data

### What Gets Created
- **Test directory**: `/E2E-Tests/e2e-{timestamp}/`
- **Small files**: 1KB text files for quick upload/download tests
- **Large files**: 10MB+ files for session management tests  
- **Directory structures**: Nested folders for directory listing tests

### File Patterns
- Text files with known content for verification
- Binary files with pattern data (not random, for consistency)
- Unicode filenames to test character handling
- Various file sizes to test different code paths

## Troubleshooting

### Common Issues

#### "E2E Testing Setup Required" Error
```
Solution: Copy your config.json to project root:
cp ~/.config/onedrive-client/config.json ./config.json
```

#### "No access token found" Error  
```
Solution: Log in first:
./onedrive-client auth login
./onedrive-client auth status
```

#### "failed to create test directory" Error
```
Possible causes:
- OneDrive quota exceeded  
- Network connectivity issues
- Account permissions problems

Check: ./onedrive-client drives quota
```

#### Tests are slow
```
Solutions:
- Increase timeout: ONEDRIVE_E2E_TIMEOUT=600s 
- Check network connection
- Run fewer tests: go test -tags=e2e -run TestE2ESetupValidation
```

### Debug Information

```bash
# Check your authentication status
./onedrive-client auth status

# Check available drives
./onedrive-client drives list

# Check quota
./onedrive-client drives quota

# List test directory (after tests)
./onedrive-client files list /E2E-Tests
```

## Manual Cleanup

If automatic cleanup fails, you can manually remove test data:

```bash
# List test directories
./onedrive-client files list /E2E-Tests

# Note: Manual deletion requires implementing delete functionality
# TODO: Add delete commands to CLI for manual cleanup
```

## Performance Expectations

- **Setup validation**: 5-10 seconds
- **Small file tests**: 10-15 seconds  
- **Large file tests**: 30-60 seconds
- **Full test suite**: 2-5 minutes
- **Network dependent**: Times vary with connection speed

## Contributing

When adding new E2E tests:

1. **Use the helper**: Always use `NewE2ETestHelper(t)` for setup
2. **Test isolation**: Don't depend on other tests' state  
3. **Cleanup**: Tests auto-cleanup, but verify in your code
4. **Error testing**: Test both success and failure scenarios
5. **Documentation**: Update this README for new test categories

## Security Notes

- **Config Safety**: `config.json` is gitignored and won't be committed
- **Token Security**: Refresh tokens are kept secure and auto-renewed
- **Account Isolation**: Tests only use designated test directories
- **No Credential Logging**: Sensitive data is never logged

## Advantages of This Approach

✅ **Simple Setup**: Just copy one file and run tests  
✅ **Same as CLI**: Uses identical authentication flow  
✅ **Token Refresh**: Automatically handles token expiration  
✅ **No Azure Setup**: No app registrations or service principals needed  
✅ **Real User Flow**: Tests exactly what users experience  
✅ **Local Development**: Works great for local testing  
✅ **Safe**: Multiple safety mechanisms protect your data

## Future Enhancements

- [ ] Add delete functionality for manual cleanup
- [ ] Performance benchmarking tests  
- [ ] Network error simulation
- [ ] Concurrent operation testing
- [ ] Visual test reports
- [ ] CI/CD integration (when service principal support is added later) 