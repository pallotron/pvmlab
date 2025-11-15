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
