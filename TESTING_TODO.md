# Plan for testing CLI commands

Building a good test suite for a CLI application built with Cobra is crucial. Here is the plan I would follow to add tests to the `pvmlab/cmd` package without making any changes yet.

## Goal

The primary goal is to test the _CLI layer_ itself, not to re-test the underlying business logic (which we've already covered in the `internal` packages). The tests will verify:

1.**Command Structure**: Ensure all commands and subcommands are correctly registered and callable (e.g., `pvmlab vm list` is a valid command path). 2.**Flag & Argument Parsing**: Confirm that flags and arguments are correctly parsed and validated (e.g., a required `--name` flag is enforced). 3.**Logic Invocation**: Verify that the correct underlying functions from the `internal` packages are called with the values parsed from flags and arguments. 4.**User Output**: Check that the command produces the expected output to the console (`stdout`) and that errors are printed correctly (`stderr`).

## Strategy

I will use a combination of standard Go testing practices and features from the Cobra library to achieve this in an isolated and repeatable way.

1.**Output Capturing**: For each test, I will redirect the command's `stdout` and `stderr` streams to an in-memory buffer (`bytes.Buffer`). This allows me to capture everything the command would print to the console and assert against its content.

2.**Command Execution in Tests**: Instead of running the compiled binary, I will directly call the `Execute()` method on the root command object within the test. I can programmatically set the arguments for each test case using `rootCmd.SetArgs([]string{"vm", "list"})`.

3.**Dependency Mocking**: The most critical part is to isolate the CLI logic from the rest of the application. The commands call functions in our `internal` packages. Since we have already refactored those packages to be testable (using exported function variables like `config.GetAppDirFunc`), I will replace these functions with mocks during the test setup. This prevents tests from having side effects (like creating real files or making network calls) and allows me to control the data the commands receive.

## Step-by-Step Implementation Plan

### Create Test Files

I will start by creating a new test file, `pvmlab/cmd/cmd_test.go`, to house the tests. As the suite grows, we can split this into multiple files like `vm_cmd_test.go`, `socketvmnet_cmd_test.go`, etc., for better organization.

### Develop a Test Helper

To avoid repetitive code, I'll create a helper function. This function will take a command (e.g., `vm`, `list`) and its arguments, set up the output buffers, execute the command, and return the captured `stdout`, `stderr`, and any error.

### Test a Simple Read-Only Command

I'll begin with `pvmlab vm list`.

- **Arrange**: In the test, I'll mock the `metadata.GetAll()` function to return a predefined list of virtual machines (e.g., `{"vm1": {...}, "vm2": {...}}`).
- **Act**: I'll use the test helper to execute the command with the arguments `["vm", "list"]`.
- **Assert**: I'll check that the command returned no error and that the captured output string contains the names of the mocked VMs ("vm1" and "vm2").

### Test a Command with Flags and Side Effects

Next, I'll tackle a more complex command like `pvmlab vm create`.

- **Arrange**: I will mock all the functions this command depends on, such as `metadata.Save`, `cloudinit.CreateISO`, and `runner.Run`.
- **Act (Success)**: I'll execute the command with valid flags, e.g., `["vm", "create", "--name", "my-test-vm"]`.
- **Assert (Success)**: I'll verify that the command returned no error and that the mocked `metadata.Save` function was called with the name "my-test-vm".
- **Act (Failure)**: I'll execute the command _without_ the required `--name` flag.
- **Assert (Failure)**: I'll verify that the command returns an error and that the captured error output contains the expected usage information (e.g., "required flag(s) 'name' not set").

### Systematically Cover All Commands

I will repeat this "Arrange, Act, Assert" pattern for the remaining commands (`vm start`, `vm stop`, `clean`, `setup`, etc.), ensuring each one has tests for both its success and failure modes.

This plan will allow us to build a robust and maintainable test suite for the entire CLI surface, giving us high confidence that the user-facing part of the application works as intended.
