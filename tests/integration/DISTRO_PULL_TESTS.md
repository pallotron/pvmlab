# Distro Pull Integration Tests

Integration tests for the `pvmlab distro pull` command.

## Quick Start

```bash
# Install dependencies
brew install p7zip  # macOS
# docker should already be installed

# Run the tests
RUN_INTEGRATION_TESTS=true go test -v ./tests/integration -run TestDistroPull -timeout 30m
```

## What Gets Tested

### `TestDistroPull_Integration`

Tests end-to-end distro pulling:

- ✅ Downloads Ubuntu 24.04 for **aarch64** (ARM64)
- ✅ Downloads Ubuntu 24.04 for **x86_64** (Intel/AMD 64-bit)
- ✅ Extracts ISO contents using 7z
- ✅ Verifies files are created in correct location
- ✅ Tests error cases (invalid distro/arch)

### Skipped Tests (Future)

- `TestDistroPull_Cancellation` - SIGINT handling
- `TestDistroPull_ResumeAbility` - Idempotency checks

## Prerequisites

| Requirement | Check Command | Install |
|-------------|---------------|---------|
| 7z | `which 7z` | `brew install p7zip` |
| docker | `which docker` | `brew install docker` or Docker Desktop |
| Network | `ping google.com` | - |
| Disk Space | `df -h` | ~5GB free |

**Note**: This test will automatically skip if 7z or docker binaries are not found. In the main integration test workflow (`.github/workflows/integration-test.yml`), this test will be skipped since Docker is not installed on the macOS runners used for VM lifecycle tests.

## Running Tests

### Local Development

```bash
# Run all distro pull tests (both architectures)
RUN_INTEGRATION_TESTS=true go test -v ./tests/integration -run TestDistroPull -timeout 45m

# Run only aarch64 tests
RUN_INTEGRATION_TESTS=true go test -v ./tests/integration -run "TestDistroPull/ubuntu.*aarch64"

# Run only x86_64 tests
RUN_INTEGRATION_TESTS=true go test -v ./tests/integration -run "TestDistroPull/ubuntu.*x86_64"

# Run with custom home directory
PVMLAB_HOME=/tmp/my-test RUN_INTEGRATION_TESTS=true go test -v ./tests/integration -run TestDistroPull

# Keep test artifacts for debugging
PVMLAB_DEBUG=true RUN_INTEGRATION_TESTS=true go test -v ./tests/integration -run TestDistroPull
```

### CI/CD

The tests run automatically via GitHub Actions in `.github/workflows/distro-pull-test.yml`:

- **Trigger**: Manual dispatch or weekly schedule (Sundays 2 AM UTC)
- **Runner**: macOS-latest
- **Timeout**: 45 minutes
- **Dependencies**: Automatically installs p7zip and uses Docker (pre-installed on GitHub macOS runners)

**Important**: This test is **separate** from the main integration test workflow (`.github/workflows/integration-test.yml`). The main workflow runs VM lifecycle tests and does not install Docker, so the distro pull test will automatically skip there.

To trigger manually:

1. Go to Actions tab in GitHub
2. Select "Distro Pull Integration Tests"
3. Click "Run workflow"

## Test Files

```shell
tests/integration/
├── distro_pull_test.go          # Integration test implementation
├── main_test.go                  # Test setup (TestMain)
└── utils.go                      # Helper functions
```

## How It Works

1. **Setup** (`TestMain`):
   - Checks `RUN_INTEGRATION_TESTS=true`
   - Creates temporary `PVMLAB_HOME`
   - Builds pvmlab binary

2. **Test Execution**:
   - Creates test-specific subdirectory
   - Runs `pvmlab distro pull --distro ubuntu-24.04 --arch aarch64`
   - Runs `pvmlab distro pull --distro ubuntu-24.04 --arch x86_64`
   - Validates output directory structure
   - Checks for extracted files

3. **Cleanup**:
   - Removes test directory (unless `PVMLAB_DEBUG=true`)

## Expected Results

After successful test:

```shell
$PVMLAB_HOME/images/ubuntu-24.04/
├── aarch64/
│   ├── kernel file(s)
│   ├── initrd file(s)
│   ├── rootfs tarball(s)
│   └── other distro-specific files
└── x86_64/
    ├── kernel file(s)
    ├── initrd file(s)
    ├── rootfs tarball(s)
    └── other distro-specific files
```

## Troubleshooting

| Error | Solution |
|-------|----------|
| `7z binary not found` | Install p7zip: `brew install p7zip` |
| `docker binary not found` | Install Docker |
| `Test skipped` | Set `RUN_INTEGRATION_TESTS=true` |
| `Download failed` | Check network connectivity |
| `No space left` | Free up ~5GB disk space |
| `Timeout` | Increase with `-timeout 60m` |

## Why Separate From Unit Tests?

Integration tests are separated because they:

- **Require external dependencies** (7z, docker, network)
- **Take significant time** (5-15 minutes per distro)
- **Download large files** (~2-4GB ISOs)
- **Are not suitable for every CI run**
- **May fail due to external factors** (network, upstream changes)

## Adding New Distro Tests

To test a new distro:

```go
{
    name:        "pull fedora-39 x86_64",
    distro:      "fedora-39",
    arch:        "x86_64",
    expectError: false,
    validateFunc: func(t *testing.T, homeDir string) {
        expectedDir := filepath.Join(homeDir, "images", "fedora-39", "x86_64")
        // Add fedora-specific validation
    },
},
```

## Performance

Approximate execution times (per architecture):

- Setup: ~30 seconds
- Download: 2-10 minutes (network dependent)
- Extract: 1-3 minutes
- **Total per architecture**: ~5-15 minutes
- **Total for all tests**: ~15-35 minutes (runs both aarch64 and x86_64)

## Notes

- Tests create isolated `PVMLAB_HOME` directories to avoid conflicts
- Downloads are not cached between test runs
- Each test run downloads fresh ISOs (separate downloads for aarch64 and x86_64)
- Both architectures are tested to ensure cross-platform compatibility
- ISO sizes vary: Ubuntu 24.04 aarch64 (~2.5GB), x86_64 (~3.8GB)
- Consider running these tests on a schedule rather than on every commit
- You can run individual architecture tests to save time during development
