# Testing Strategy for CLI Commands

This document outlines the testing strategy for the `pvmlab` CLI. It covers both the existing end-to-end integration tests and the plan for adding complementary, fast-running unit tests.

## Current State: Integration Testing

A full, end-to-end integration test exists in `tests/integration/main_test.go`.

**Goal:** To verify that all components of the application work together correctly in a real-world scenario, covering the entire lifecycle of a provisioner VM.

**What it Covers:**

- The "happy path" for the entire VM lifecycle: `setup`, `vm create`, `vm start`, `vm list`, `vm stop`, and `vm clean`.
- Correct parsing of arguments and flags (e.g., `--role`, `<vm-name>`).
- Real-world side effects: creating disk images, launching a real QEMU process, and waiting for the SSH port to become available.
- Dynamic SSH port allocation and usage across multiple commands (`vm start`, `vm shell`, `provisioner docker *`).

This test provides the highest level of confidence that the application is functional. However, it is slow and does not cover specific failure cases.

## Next Steps: Complementary CLI Unit Tests

To cover failure modes and edge cases in a fast and isolated manner, we will add unit tests for the `pvmlab/cmd` package.

### Goal

The primary goal is to test the _CLI layer_ itself in isolation, focusing on scenarios not covered by the integration test. The tests will verify:

1. **Flag & Argument Validation**: Confirm that errors are produced for missing or invalid flags and arguments (e.g., `vm create` without a name, `provisioner create` without `--ip`).
2. **Error Handling**: Ensure that errors from the underlying `internal` packages are caught and presented to the user correctly.
3. **User Output**: Check that help text and error messages are printed as expected.

### Strategy

We will use a combination of standard Go testing practices and features from the Cobra library.

1. **Output Capturing**: For each test, we will redirect the command's `stdout` and `stderr` streams to an in-memory buffer (`bytes.Buffer`) to assert against its content.

2. **Command Execution in Tests**: We will directly call the `Execute()` method on the root command object within the test, setting arguments programmatically with `rootCmd.SetArgs()`.

3. **Dependency Mocking**: We will mock the functions in our `internal` packages. This prevents tests from having side effects (like creating real files) and allows us to simulate specific failure conditions (e.g., `metadata.Save` returning an error).

### Example Implementation (`pvmlab provisioner create`)

- **Arrange**: Mock the `metadata.Save` function to return an error.
- **Act**: Execute the command with valid flags: `["provisioner", "create", "my-test-prov", "--ip", "192.168.1.1/24"]`.
- **Assert**: Verify that the command returned an error and that the captured `stderr` contains the expected error message.
- **Act (Failure)**: Execute the command _without_ the required `--ip` flag.
- **Assert (Failure)**: Verify that the command returns an error and that the captured error output contains the expected usage information.

This two-pronged testing strategy (broad integration tests + targeted unit tests) will provide robust test coverage for the entire application.
