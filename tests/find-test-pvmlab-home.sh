#!/bin/bash
set -e
# This script finds the PVMLAB_HOME directory for a running integration test.
# It searches for temporary directories in /tmp that contain a VM pid file.

for dir in /tmp/pvmlab_test_*; do
    if [ -d "$dir" ]; then
        if [ -f "$dir/.pvmlab/pids/test-provisioner.pid" ] || [ -f "$dir/.pvmlab/pids/test-client.pid" ]; then
            echo "$dir"
            exit 0
        fi
    fi
done

echo "No running integration test VM found." >&2
exit 1
