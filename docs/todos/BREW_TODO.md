# Homebrew Formula Implementation Plan

This document outlines the plan to create and maintain a Homebrew formula for `pvmlab` to simplify installation on macOS.

## Goal

The primary goal is to provide a seamless, standard installation method for macOS users via the `brew` command. This will handle dependencies, install the necessary binaries, and set up the required launchd service.

## Status

- [x] Implement `pvmlab system setup-launchd` command.
- [x] Implement `pvmlab.rb` formula.
- [x] Move `pvmlab.rb` to `brew/` directory.
- [x] Create `homebrew-pvmlab` repository.
- [ ] Set up GitHub Action to automatically update the formula on release.
- [ ] Add installation instructions to `README.md`.

## Detailed Tasks

### 1. Formula (`pvmlab.rb`) Implementation

The formula will be hosted in a new, dedicated public GitHub repository named `homebrew-pvmlab`.

#### Metadata and Source

- **Metadata:** `desc`, `homepage`, `license`.
- **Source:** A `url` pointing to a specific versioned release tarball (e.g., `.../archive/refs/tags/v0.1.0.tar.gz`) and a corresponding `sha256` checksum.

#### Dependencies

The formula must declare all necessary dependencies.

```ruby
depends_on "go" => :build
depends_on "qemu"
depends_on "cdrtools"
depends_on "socat"
depends_on "socket_vmnet"
```

#### `install` Block

The `install` block will perform the following actions:

1. **Compile `pvmlab`:**
    - Compile the main `pvmlab` binary from source, injecting the version number using `-ldflags`.
    - Install the compiled `pvmlab` binary to `bin/`.

2. **Install Supporting Files:**
    - Install the `socket_vmnet_wrapper.sh` script to the formula's `libexec` directory (`libexec.install "launchd/socket_vmnet_wrapper.sh"`).
    - Install the `io.github.pallotron.pvmlab.socket_vmnet.plist` launchd service file into the formula's prefix for later use (`prefix.install "launchd/io.github.pallotron.pvmlab.socket_vmnet.plist"`).

3. **Install Shell Completions:**
    - Generate and install shell completions for `bash` and `zsh`.

#### `post_install` or `caveats` Block

Since installing and loading a launchd service requires `sudo`, Homebrew cannot do this automatically. The formula must provide clear instructions to the user.

A `caveats` message is the most appropriate way to handle this. It will instruct the user to run a command to finalize the setup. We can add a dedicated command to `pvmlab` for this purpose, for example `pvmlab system setup-launchd`.

The `caveats` message would look like this:

```text
To complete the installation and set up the networking service, run the following command:

  sudo pvmlab system setup-launchd
```

### 2. `pvmlab system setup-launchd` Command

This new command needs to be created within the `pvmlab` CLI. It will perform the necessary privileged operations:

1. Create the `/opt/pvmlab/libexec/` directory.
2. Copy the `socket_vmnet_wrapper.sh` from the Homebrew `libexec` directory to `/opt/pvmlab/libexec/`.
3. Copy the `.plist` file from the Homebrew prefix to `/Library/LaunchDaemons/`.
4. Load and start the launchd service using `launchctl`.

This approach keeps the Homebrew formula clean and informs the user about the necessary `sudo` command, which is standard practice for formulas requiring elevated permissions.

### 3. GitHub Action for Automatic Updates

To automate the release process, we will add a job to `.github/workflows/release.yml` that updates the Homebrew formula in the `homebrew-pvmlab` repository whenever a new release is created.

**Requirements:**

- Create `homebrew-pvmlab` repository.
- Create a PAT with `repo` scope.
- Add PAT to repository secrets as `GH_PAT_HOMEBREW`.
- Update workflow to use `mislav/bump-homebrew-formula-action`.

### 4. User Instructions

Update `README.md` to instruct users on how to install `pvmlab` using the custom tap:

```bash
brew tap pallotron/pvmlab
brew install pvmlab
sudo pvmlab system setup-launchd
```
