#!/bin/sh
# This script builds a custom initrd for pvmlab.
# It is executed inside a containerized build environment.

set -e

# Arguments from Makefile
TARGET_ARCH=$1
INSTALLER_NAME=$2
OUTPUT_DIR=$3
PACKAGES=$4
TOOLS=$5

echo "==> [Container] Starting custom initrd build for ${TARGET_ARCH}..."

STAGE_DIR="/work/${OUTPUT_DIR}/initrd_root_${TARGET_ARCH}"
rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}/bin" "${STAGE_DIR}/dev" "${STAGE_DIR}/proc" "${STAGE_DIR}/sys" "${STAGE_DIR}/etc/terminfo/x" "${STAGE_DIR}/etc/terminfo/v" "${STAGE_DIR}/tmp"
chmod 1777 "${STAGE_DIR}/tmp"

# Copy the custom init script
cp "/work/init" "${STAGE_DIR}/init"
chmod +x "${STAGE_DIR}/init"

echo "==> [Container] Permissions of /init after chmod:"
ls -l "${STAGE_DIR}/init"

# Copy busybox explicitly
BUSYBOX_PATH=$(which busybox)
cp "${BUSYBOX_PATH}" "${STAGE_DIR}/bin/busybox"
chmod +x "${STAGE_DIR}/bin/busybox"

# Copy the os-installer
cp "/work/${OUTPUT_DIR}/${INSTALLER_NAME}" "${STAGE_DIR}/bin/os-installer"

# Needed by bash
cp "/etc/terminfo/x/xterm" "${STAGE_DIR}/etc/terminfo/x/xterm"
cp "/etc/terminfo/v/vt100" "${STAGE_DIR}/etc/terminfo/v/vt100"

# Copy the udhcpc default script
mkdir -p "${STAGE_DIR}/usr/share/udhcpc"
cp "/usr/share/udhcpc/default.script" "${STAGE_DIR}/usr/share/udhcpc/default.script"

# Copy dynamic linker
DYNAMIC_LINKER_PATH="/lib/ld-musl-${TARGET_ARCH}.so.1"
mkdir -p "${STAGE_DIR}$(dirname "${DYNAMIC_LINKER_PATH}")"
cp "${DYNAMIC_LINKER_PATH}" "${STAGE_DIR}${DYNAMIC_LINKER_PATH}"
chmod +x "${STAGE_DIR}${DYNAMIC_LINKER_PATH}"

# Copy tools and their dependencies
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

echo "==> [${TARGET_ARCH}] Installing busybox symlinks..."
(cd "${STAGE_DIR}/bin" && for P in $(./busybox --list); do ln -s busybox "$P"; done)

# Ensure /bin/reboot is a relative symlink to busybox
ln -sf busybox "${STAGE_DIR}/bin/reboot"

echo "==> [${TARGET_ARCH}] Contents of staging directory (${STAGE_DIR}):"
find "${STAGE_DIR}" -print

echo "==> [${TARGET_ARCH}] Creating initramfs archive..."
mkdir -p "/work/${OUTPUT_DIR}/${TARGET_ARCH}"
cd "${STAGE_DIR}"
find . -print0 \
| cpio --null -ov --format=newc \
| gzip -9 > "/work/${OUTPUT_DIR}/${TARGET_ARCH}/initrd.gz"

echo "==> [${TARGET_ARCH}] Build complete."
echo "==> Output initrd: /work/${OUTPUT_DIR}/${TARGET_ARCH}/initrd.gz"
