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
2. Valid Loki credentials (username, password, org ID)
3. A test PipelineRun name (optional, for full testing)

### Environment Variables

Set the following environment variables:

```bash
export LOKI_URL="https://loki.launchpad.kr"
export LOKI_USERNAME="your-loki-username"
export LOKI_PASSWORD="your-loki-password"
export LOKI_ORG_ID="fake"  # or your org ID
export TEST_PIPELINE_RUN_NAME="image-build-push-run-xyz123"  # Optional: actual PipelineRun name
```

### Run Integration Tests

```bash
# Run all integration tests (including Loki)
go test -v ./test/integration/...

# Run only Loki integration tests
go test -v ./test/integration/ -run TestLoki

# Skip integration tests (short mode)
go test -short ./test/integration/...
```

## Test Behavior

### With Environment Variables Set

If `LOKI_URL` and credentials are configured, the tests will:
1. Connect to the real Loki instance
2. Attempt to stream logs for the specified PipelineRun
3. Validate that the stream can be opened and read from

### Without Environment Variables

If Loki credentials are not set, the tests will:
- Automatically skip with a message: `"LOKI_URL not set, skipping Loki integration test"`
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
