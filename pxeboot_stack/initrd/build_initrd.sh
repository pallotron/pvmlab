#!/bin/sh
# This script is the main entry point for building the initrd.
# It is executed inside a containerized build environment orchestrated by the Makefile.

set -e

# Arguments from Makefile
TARGET_ARCH=$1
GO_ARCH=$2
MUSL_ARCH=$3
INSTALLER_NAME=$4
OUTPUT_DIR=$5
UROOT_SRC_DIR=$6
PACKAGES=$7
TOOLS=$8
GO_VERSION=$9

# --- The following runs inside the Docker container ---

echo "==> [Container] Starting initrd build for ${TARGET_ARCH}..."
# set -x # Uncomment for debugging

echo "==> [${TARGET_ARCH}] Building installer..."
cd "/work/installer" && GOWORK=off go mod tidy && GOWORK=off go mod vendor && GOARCH=${GO_ARCH} go build -o "/work/${OUTPUT_DIR}/${INSTALLER_NAME}" .

echo "==> [${TARGET_ARCH}] Populating staging directory..."
STAGE_DIR="/work/${OUTPUT_DIR}/initrd_root_${TARGET_ARCH}"
rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}/bin" "${STAGE_DIR}/bbin" "${STAGE_DIR}/etc/terminfo/x" "${STAGE_DIR}/etc/terminfo/v"

cp "/work/uinit" "${STAGE_DIR}/uinit"
cp "/work/${OUTPUT_DIR}/${INSTALLER_NAME}" "${STAGE_DIR}/bbin/os-installer"

# needed by bash
cp "/etc/terminfo/x/xterm" "${STAGE_DIR}/etc/terminfo/x/xterm"
cp "/etc/terminfo/v/vt100" "${STAGE_DIR}/etc/terminfo/v/vt100"

DYNAMIC_LINKER_PATH="/lib/ld-musl-${MUSL_ARCH}.so.1"
mkdir -p "${STAGE_DIR}$(dirname "${DYNAMIC_LINKER_PATH}")"
cp "${DYNAMIC_LINKER_PATH}" "${STAGE_DIR}${DYNAMIC_LINKER_PATH}"

for tool in $TOOLS; do
    echo "==> [${TARGET_ARCH}] Installing tool: ${tool}"
    TOOL_PATH=$(which "${tool}")
    cp "${TOOL_PATH}" "${STAGE_DIR}/bin/$(basename "${TOOL_PATH}")"
    # Copy shared library dependencies
    for lib in $(ldd "${TOOL_PATH}" | awk 'NF == 4 {print $3}; NF == 2 {print $1}'); do
        if [ -n "${lib}" ] && [ -f "${lib}" ]; then
            echo "    -> Found and copying library: ${lib}"
            mkdir -p "${STAGE_DIR}$(dirname "${lib}")"
            cp "${lib}" "${STAGE_DIR}${lib}"
        fi
    done
done

echo "    -> Ensuring /bin/reboot is a relative symlink to busybox"
ln -sf busybox "${STAGE_DIR}/bin/reboot"

echo "==> [${TARGET_ARCH}] Contents of staging directory (${STAGE_DIR}):"
find "${STAGE_DIR}" -print

echo "==> [${TARGET_ARCH}] Building u-root..."
cd "/work/${UROOT_SRC_DIR}" && go build .

echo "==> [${TARGET_ARCH}] Building initrd..."
mkdir -p "/work/${OUTPUT_DIR}/${TARGET_ARCH}"
./u-root \
    -o "/work/${OUTPUT_DIR}/${TARGET_ARCH}/initrd" \
    -build=bb \
    -uinitcmd=/uinit \
    -files "${STAGE_DIR}:." \
    core boot

echo "==> [${TARGET_ARCH}] Compressing initrd..."
gzip -f "/work/${OUTPUT_DIR}/${TARGET_ARCH}/initrd"

echo "==> [${TARGET_ARCH}] Build complete."
echo "==> Output initrd: /work/${OUTPUT_DIR}/${TARGET_ARCH}/initrd.gz"
