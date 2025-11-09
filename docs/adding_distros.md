# How to Add a New Distribution

This guide outlines the process for adding support for a new Linux distribution to `pvmlab`. The system is designed to be extensible, but each new distro requires some specific information to be gathered first.

## Step 1: Gather Distribution Information

Before making any code changes, you need to find the following information for the new distribution:

1. **ISO URLs:** Find the official download URLs for the **server** or **netinstall** ISO images for both `aarch64` (ARM64) and `x86_64` (AMD64) architectures.
2. **Kernel Path:** Determine the exact path to the kernel file (usually named `vmlinuz`) *inside* the ISO.

### How to Find the Kernel and Initrd Paths

The most reliable way to find these paths is to download the ISO and inspect its contents.

1. **Download the ISO:**

    ```bash
    # Create a temporary directory
    mkdir /tmp/iso_inspect
    # Download the ISO (example for Fedora)
    curl -L -o /tmp/iso_inspect/distro.iso <URL_of_the_ISO>
    ```

2. **List the ISO Contents:**
    Use the `7z l` command to list the files inside the ISO without extracting the whole thing. Look for files named `vmlinuz` and `initrd.img` (or similar). They are often located in directories like `/images/pxeboot/`, `/casper/`, or `/boot/`.

    ```bash
    7z l /tmp/iso_inspect/distro.iso | grep 'vmlinuz'
    7z l /tmp/iso_inspect/distro.iso | grep 'initrd'
    ```

    Note down the full paths as they appear in the listing (e.g., `images/pxeboot/vmlinuz`).

## Step 2: Update the Configuration

Once you have the necessary information, you can add the new distribution to the configuration.

1. **Edit `~/.pvmlab/distros.yaml`:**
    Open the `distros.yaml` file in your `~/.pvmlab` directory and add a new entry for your distribution. This file is loaded by `pvmlab` at runtime.

    ```yaml
    - name: my-distro-1.0 # Unique, user-facing identifier (e.g., for the --distro flag)
      distro_name: my-distro # The "family" name (e.g., ubuntu, fedora). This MUST match a case in the Extractor factory.
      version: "1.0"
      arch:
        aarch64:
          iso_url: "https://example.com/my-distro-1.0-aarch64.iso"
          iso_name: "my-distro-1.0-aarch64.iso"
          kernel_file: "path/inside/iso/to/vmlinuz"
        x86_64:
          iso_url: "https://example.com/my-distro-1.0-x86_64.iso"
          iso_name: "my-distro-1.0-x86_64.iso"
          kernel_file: "path/inside/iso/to/vmlinuz"
    ```

    **Understanding `name` vs. `distro_name`:**

    - **`name`**: This is the unique identifier for a specific version of a distribution. It's what you will use with the `--distro` flag in `pvmlab` commands.
    - **`distro_name`**: This is the generic "family" name for the distribution (e.g., `ubuntu`, `fedora`, `debian`). This field is critical as it tells `pvmlab` which `Extractor` logic to use for processing the ISO. For example, both `ubuntu-22.04` and `ubuntu-24.04` would have a `distro_name` of `ubuntu`.

    **Note:** If you intend to contribute this new distribution definition back to the `pvmlab` project, you should **also** edit the `internal/config/distros.yaml` file in the source code.

## Step 3: Implement the Extractor

The `Extractor` is a Go interface that handles the logic for extracting boot files from a specific distribution's ISO.

1. **Create a New Go File:**
    In the `internal/distro/` directory, create a new file named after your distribution (e.g., `mydistro.go`).

2. **Implement the `Extractor` Interface:**
    Create a struct for your extractor and implement the `ExtractKernelAndModules` method.

    ```go
    package distro

    import (
        "fmt"
        "pvmlab/internal/config"
    )

    // MyDistroExtractor implements the Extractor interface.
    type MyDistroExtractor struct{}

    func (e *MyDistroExtractor) ExtractKernelAndModules(cfg *config.Config, distroInfo *config.ArchInfo, isoPath, distroPath string) error {
        //
        // Add the logic here to extract the kernel and initrd.
        // For many distros, this will be a simple extraction of the two files.
        //
        // For example, for a distro that provides a self-contained initrd:
        // 1. Extract distroInfo.KernelFile to distroPath/vmlinuz
        // 2. Extract distroInfo.InitrdFile to distroPath/modules.cpio.gz (renaming is important)
        //
        // If the distro requires building the initrd (like Ubuntu), you will need
        // to implement that more complex logic here.
        //
        return fmt.Errorf("extraction for my-distro is not yet implemented")
    }
    ```

3. **Register the Extractor:**
    Open `internal/distro/extract.go` and add your new extractor to the `NewExtractor` factory function.

    ```go
    // In internal/distro/extract.go
    func NewExtractor(distroName string) (Extractor, error) {
        switch distroName {
        case "ubuntu":
            return &UbuntuExtractor{}, nil
        case "fedora":
            return &FedoraExtractor{}, nil
        case "my-distro": // Add your new distro here
            return &MyDistroExtractor{}, nil
        default:
            return nil, fmt.Errorf("no extractor available for distribution: %s", distroName)
        }
    }
    ```

## Step 4: Build and Test

After implementing the extractor, you can build `pvmlab` and test your new distribution.

1. **Build the CLI:**

    ```bash
    make install-pvmlab
    ```

2. **Pull the Distro:**

    ```bash
    pvmlab distro pull --distro my-distro-1.0 --arch <arch>
    ```

3. **Create a VM:**

    ```bash
    pvmlab vm create my-test-vm --pxeboot --distro my-distro-1.0 --arch <arch>
    ```

If everything is configured correctly, the `distro pull` command should successfully download and extract the boot files, and you will be able to create PXE-booted VMs with your new distribution.
