# Release Process

## Automated Release Steps

When you push a tag matching `v*.*.*`, GitHub Actions will automatically:

1. Create a GitHub release
2. Build and upload the AMD64 provisioner image (`provisioner-custom.amd64.qcow2`)
3. Build initrd files for PXE boot
4. Build and push the PXE boot container to GitHub Container Registry

## Manual ARM64 Image Build

ARM64 provisioner images must be built and uploaded manually because GitHub's ARM64 runners require a paid account.

### Prerequisites

- ARM64 machine (e.g., Apple Silicon Mac)
- Docker installed
- GitHub CLI (`gh`) installed and authenticated

### Steps

1. **Checkout the release tag:**

   ```bash
   git checkout v0.0.x  # Replace with your version
   ```

2. **Build the ARM64 image:**

   ```bash
   cd image_builder
   ./build-image.sh arm64
   ```

   This will create `output/provisioner-custom.arm64.qcow2`

3. **Upload to the release:**

   ```bash
   gh release upload v0.0.x ./output/provisioner-custom.arm64.qcow2 --clobber
   ```

   The `--clobber` flag will replace the asset if it already exists.

4. **Verify the upload:**

   ```bash
   gh release view v0.0.x
   ```

## Complete Release Checklist

- [ ] Tag pushed (triggers automated build)
- [ ] AMD64 image built and uploaded (automated)
- [ ] PXE boot container published (automated)
- [ ] ARM64 image built locally
- [ ] ARM64 image uploaded to release
- [ ] Verify both images are available in the release
- [ ] Test download and deployment of both architectures

## Manual PXE Boot Container Build and Push

If the automated GitHub Actions workflow fails or you need to push a container manually without creating a release tag:

### Prerequisites

- Docker with buildx support
- GitHub Personal Access Token with `write:packages` scope
- 1Password CLI (`op`) if storing token in 1Password

### Steps

1. **Sign in to 1Password (if using):**

   ```bash
   eval $(op signin)
   ```

2. **Log in to GitHub Container Registry:**

   ```bash
   # Using 1Password CLI
   op read "op://Private/GitHub Personal Access Token/password" | docker login ghcr.io -u YOUR_USERNAME --password-stdin

   # Or using environment variable
   export GITHUB_TOKEN=your_token_here
   echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_USERNAME --password-stdin
   ```

   Replace "GitHub Personal Access Token" accordingly to your 1password vault.

3. **Build initrds:**

   ```bash
   git submodule update --init --recursive
   make -C pxeboot_stack/initrd all
   ```

4. **Build boot_handler binaries:**

   ```bash
   make -C pxeboot_stack build-boot-handler
   ```

5. **Build and push the container:**

   ```bash
   docker buildx build \
     --platform linux/amd64,linux/arm64 \
     --push \
     --tag ghcr.io/pallotron/pvmlab/pxeboot_stack:manual \
     --tag ghcr.io/pallotron/pvmlab/pxeboot_stack:latest \
     ./pxeboot_stack
   ```

   You can replace `manual` with any tag you prefer (e.g., `v0.0.3`, `dev`, etc.)

## Example

```bash
# Create and push a new release tag
git tag v0.0.3
git push origin v0.0.3

# Wait for GitHub Actions to complete the automated builds

# Build and upload ARM64 manually
git checkout v0.0.3
cd image_builder
./build-image.sh arm64
gh release upload v0.0.3 ./output/provisioner-custom.arm64.qcow2

# Verify
gh release view v0.0.3
```
