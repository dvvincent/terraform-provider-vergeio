# Testing the VergeIO Terraform Provider

This document describes how to run acceptance tests for the `terraform-provider-vergeio`.

## Overview

The provider uses [terraform-plugin-testing](https://github.com/hashicorp/terraform-plugin-testing) for automated acceptance tests. These tests execute real Terraform operations (`plan`, `apply`, `destroy`) against a live VergeOS instance.

## Prerequisites

1. **Go 1.23+** installed
2. **Access to a VergeOS instance** (test or development environment recommended)
3. **Credentials** (either API token or username/password)

## Environment Variables

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `VERGEOS_HOST` | VergeOS API endpoint | `https://192.168.1.138` |
| `VERGEOS_USERNAME` | Username for auth (if not using token) | `admin` |
| `VERGEOS_PASSWORD` | Password for auth (if not using token) | `password123` |
| `VERGEOS_TOKEN` | API token (alternative to user/pass) | `abc123...` |

### Optional (Resource-Specific)

| Variable | Description | Required For |
|----------|-------------|--------------|
| `VERGEOS_TEST_VNET_ID` | VNet ID for firewall rule tests | `TestAccVNetRule_*` |
| `VERGEOS_TEST_TARGET_IP` | Target IP for NAT tests | `TestAccVNetRule_nat` |

## Running Tests

### Using the Helper Script

```bash
# Set credentials
export VERGEOS_HOST="https://192.168.1.138"
export VERGEOS_USERNAME="admin"
export VERGEOS_PASSWORD="your-password"

# Run all tests
./scripts/run-tests.sh

# Run specific tests
./scripts/run-tests.sh TestAccTag        # Tag tests only
./scripts/run-tests.sh TestAccVNetRule   # VNet rule tests only

# Verbose output
./scripts/run-tests.sh TestAccTag -v
```

### Using Go Directly

```bash
# Run all acceptance tests
TF_ACC=1 go test ./internal/provider/... -timeout=30m

# Run a specific test
TF_ACC=1 go test ./internal/provider/tags/... -run=TestAccTag_basic -v

# Run with race detection (slower but catches concurrency issues)
TF_ACC=1 go test ./internal/provider/... -race -timeout=60m
```

## Test Categories

### Tag Tests (`internal/provider/tags/tag_test.go`)

These tests don't require any special setup beyond credentials:

- `TestAccTagCategory_basic` - Create, update, import a tag category
- `TestAccTag_basic` - Create, update, import a tag with its category

### VNet Rule Tests (`internal/provider/network/vnet_rule_test.go`)

Require `VERGEOS_TEST_VNET_ID` to be set:

```bash
# Find a VNet ID to use for testing
curl -k -u admin:password https://192.168.1.138/api/v4/vnets | jq '.[0].$key'

# Set it
export VERGEOS_TEST_VNET_ID=3
```

Available tests:
- `TestAccVNetRule_basic` - Create accept rule, update port, verify import
- `TestAccVNetRule_nat` - Create NAT/translate rule
- `TestAccVNetRule_drop` - Create drop rule for ICMP

### Network Tests (`internal/provider/network/network_test.go`)

These tests don't require any special setup beyond credentials:

- `TestAccNetwork_basic` - Create, update, import an internal network
- `TestAccNetwork_withDHCP` - Create a network with DHCP enabled
- `TestAccNetwork_internalType` - Create a network with explicit internal type
- `TestAccNetwork_powerState` - Create a network with power state management

## Writing New Tests

### Test Structure

Each test follows this pattern:

```go
func TestAccResourceName_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Step 1: Create
            {
                Config: testAccResourceConfig("initial-value"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("vergeio_resource.test", "field", "initial-value"),
                ),
            },
            // Step 2: Update
            {
                Config: testAccResourceConfig("updated-value"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("vergeio_resource.test", "field", "updated-value"),
                ),
            },
            // Step 3: Import
            {
                ResourceName:      "vergeio_resource.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

### Common Checks

```go
// Verify an attribute has a specific value
resource.TestCheckResourceAttr(resourceName, "field", "expected-value")

// Verify an attribute is set (any value)
resource.TestCheckResourceAttrSet(resourceName, "id")

// Verify an attribute exists with a pattern
resource.TestMatchResourceAttr(resourceName, "urn", regexp.MustCompile(`urn:vergeio:.*`))

// Verify resource count
resource.TestCheckResourceAttr(resourceName, "items.#", "3")
```

## Troubleshooting

### "VERGEOS_HOST must be set"

Ensure you've exported the environment variables before running tests:

```bash
export VERGEOS_HOST="https://your-vergeos-host"
export VERGEOS_USERNAME="admin"
export VERGEOS_PASSWORD="password"
```

### "Login required" or 401 errors

- Verify credentials are correct
- Check if the token has expired (tokens typically expire after 24 hours)
- Try switching from token to username/password authentication

### Tests hang or timeout

- Default timeout is 30 minutes; increase with `-timeout=60m`
- Check VergeOS system logs for stuck operations
- Verify network connectivity to the VergeOS host

### "Provider produced inconsistent result"

This usually means the API returns a different value than what Terraform expects:

1. Check if VergeOS modifies the field (e.g., normalizing case, adding defaults)
2. Update the resource's `Read` function to handle the transformation
3. Consider using `ImportStateVerifyIgnore` for computed fields

## CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: Acceptance Tests

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Run acceptance tests
        env:
          TF_ACC: "1"
          VERGEOS_HOST: ${{ secrets.VERGEOS_HOST }}
          VERGEOS_USERNAME: ${{ secrets.VERGEOS_USERNAME }}
          VERGEOS_PASSWORD: ${{ secrets.VERGEOS_PASSWORD }}
        run: go test ./internal/provider/tags/... -v -timeout=30m
```

**Note:** Only run resource-light tests (like tags) in CI to avoid resource conflicts and long run times.
