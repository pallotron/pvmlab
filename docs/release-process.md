# Release Process

## Automated Release Steps

When you push a tag matching `v*.*.*`, GitHub Actions will automatically:

1. Create a GitHub release
2. Build and upload the AMD64 provisioner image (`provisioner-custom.amd64.qcow2`)
3. Build and upload the ARM64 provisioner image (`provisioner-custom.arm64.qcow2`)
4. Build initrd files for PXE boot
5. Build and push the PXE boot container to GitHub Container Registry

## Complete Release Checklist

- [ ] Tag pushed (triggers automated build)
- [ ] AMD64 image built and uploaded (automated)
- [ ] ARM64 image built and uploaded (automated)
- [ ] PXE boot container published (automated)
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

# Verify
gh release view v0.0.3
```
