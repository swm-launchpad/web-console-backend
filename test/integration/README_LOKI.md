# Loki Integration Test

This directory contains integration tests for the Loki log streaming functionality.

## Overview

The Loki integration test validates the WebSocket-based log streaming from Grafana Loki. It tests:
- Basic connectivity to Loki
- Streaming logs for a Tekton PipelineRun
- Task filtering (excluding specific tasks like `ecr-repository-check`)
- Error handling for non-existent PipelineRuns

## Running the Tests

### Prerequisites

1. Access to a Grafana Loki instance
2. Valid Loki credentials configured in `.env` file
3. A test PipelineRun name (optional, for full testing)

### Configuration

The integration tests automatically load Loki configuration from the `.env` file in the project root.

Ensure the following variables are set in your `.env` file:

```bash
# Loki Configuration (Log Aggregation)
LOKI_URL=https://loki.launchpad.kr
LOKI_USERNAME=your-loki-username
LOKI_PASSWORD=your-loki-password
LOKI_ORG_ID=fake

# Optional: for testing with a specific PipelineRun
TEST_PIPELINE_RUN_NAME=image-build-push-run-xyz123
```

### Run Integration Tests

```bash
# Run all integration tests (including Loki)
# The tests will automatically load .env file
go test -v ./test/integration/...

# Run only Loki integration tests
go test -v ./test/integration/ -run TestLoki

# Skip integration tests (short mode)
go test -short ./test/integration/...

# Or simply use make check (runs all tests including integration tests)
make check
```

## Test Behavior

### With `.env` File Configured

If `LOKI_URL` and credentials are configured in `.env`, the tests will:
1. Automatically load configuration from `.env` file
2. Connect to the real Loki instance
3. Attempt to stream logs for the specified PipelineRun
4. Validate that the stream can be opened and read from

### Without `.env` File or Missing Loki Configuration

If `.env` file is missing or Loki credentials are not set, the tests will:
- Automatically skip with a message: `"LOKI_URL not set in .env file, skipping Loki integration test"`
- This allows the test suite to run in CI/CD without requiring Loki access

## Notes

- These tests use `testing.Short()` to skip in short mode
- The tests will timeout after 10 seconds when reading from the stream
- If no logs are available, the test will still pass (empty stream is valid)
- Task filtering (`excludeTasks`) is tested with `["ecr-repository-check"]`

## Troubleshooting

### Connection Refused

```
Error: dial tcp: connection refused
```

**Solution:** Check that `LOKI_URL` is correct and the Loki server is accessible.

### Unauthorized (401)

```
Error: unexpected status: 401 Unauthorized
```

**Solution:** Verify `LOKI_USERNAME` and `LOKI_PASSWORD` are correct.

### No Logs Returned

If the test passes but no logs are shown:
- The PipelineRun may not exist
- The PipelineRun may not have generated logs yet
- The task filter may be excluding all tasks

This is expected behavior for non-existent or completed PipelineRuns.
