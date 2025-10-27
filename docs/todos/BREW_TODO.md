# Homebrew Formula Implementation Plan

This document outlines the plan to create and maintain a Homebrew formula for `pvmlab` to simplify installation on macOS.

## Goal

The primary goal is to provide a seamless, standard installation method for macOS users via the `brew` command. This will handle dependencies, install the necessary binaries, and guide the user through the required one-time privileged setup in a simple and secure manner.

## Chosen Approach: Formula with a Post-Install Helper Script

We will adopt the "Helper Script" method. This approach offers the best balance of a simplified user experience and manageable developer effort. It avoids the complexity of creating a full `.pkg` installer while still reducing the post-install steps to a single command for the user.

### Key Tasks

#### 1. Create the `pvmlab-post-install.sh` Helper Script

This script will encapsulate all the setup steps that require `sudo`.

- **Purpose:** To be the single entry point for users to authorize the installation of privileged components.
- **Location:** The script will be created in the `launchd/` directory of the main repository.
- **Functionality:** The script must perform the following actions:
  1. Accept the Homebrew formula's installation prefix (e.g., `/opt/homebrew/opt/pvmlab`) as an argument to locate the necessary files.
  2. Install the `socket_vmnet` binary to `/opt/socket_vmnet/bin/`.
  3. Set the correct ownership (`root:staff`) and permissions (`555`, `u+s`) on the `socket_vmnet` binary.
  4. Install the `socket_vmnet_wrapper.sh` script to `/opt/pvmlab/libexec/`.
  5. Install the `io.github.pallotron.pvmlab.socket_vmnet.plist` launchd service file to `/Library/LaunchDaemons/`.
  6. Unload any existing version of the launchd service before loading the new one to ensure clean upgrades.
  7. Bootstrap and enable the new launchd service using `launchctl`.
  8. Provide clear, user-friendly output at each step.

#### 2. Create the `pvmlab.rb` Homebrew Formula

This Ruby script will define how Homebrew builds and installs `pvmlab`.

- **Location:** This formula will be hosted in a new, dedicated public GitHub repository named `homebrew-pvmlab`. The file will reside at `Formula/pvmlab.rb`.
- **Content:** The formula will contain:

  1. **Metadata:** `desc`, `homepage`, `license`.
  2. **Source:** A `url` pointing to a specific versioned release tarball (e.g., `.../archive/refs/tags/v0.1.0.tar.gz`) and a corresponding `sha256` checksum.
  3. **Dependencies:** `depends_on "go" => :build` and `depends_on "qemu"`.
  4. **`install` Block:**
     - Compile the main `pvmlab` binary from source, injecting the version number using `-ldflags`.
     - Install the compiled `pvmlab` binary to `bin/`.
     - `cd` into the `socket_vmnet` directory and run `make all`.
     - Install the non-privileged `socket_vmnet_client` to `bin/`.
     - Install the privileged components (`socket_vmnet`, `socket_vmnet_wrapper.sh`, and the `.plist` file) into the formula's `prefix` so the post-install script can access them.
     - Install the `pvmlab-post-install.sh` script into `bin/`.
     - Install shell completions (`bash`, `zsh`).
  5. **`caveats` Block:**

     - Provide a clear, simple message instructing the user to run the single post-install command:

     To complete the installation, you must run the post-install script with sudo:

     ```shell
       sudo pvmlab-post-install /opt/homebrew/opt/pvmlab
     ```

  6. **`test` Block:**
     - Include a simple test to verify the binary runs and reports the correct version (e.g., `shell_output("#{bin}/pvmlab --version")`).

#### 3. Document the User Installation Process

Update the main `README.md` and any installation documentation with the new, simplified instructions.

- **Instructions:**
  1. Tap the formula repository: `brew tap pallotron/pvmlab`
  2. Install the formula: `brew install pvmlab`
  3. Run the post-install command as instructed by the installer's output.

## Alternative Considered

- **Homebrew Cask with `.pkg` Installer:** This provides a fully native, GUI-driven installation. While it offers the best user experience, it was deemed overly complex for the current stage of the project due to the high overhead of creating, signing, and notarizing a macOS installer package. The helper script approach provides a comparable level of simplicity for the user with significantly less developer effort.
