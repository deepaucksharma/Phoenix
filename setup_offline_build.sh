#\!/bin/bash
# This script has been moved to scripts/setup/setup-offline-build.sh
# Redirecting to the new location for backwards compatibility
# This file will be removed in a future release

echo "Warning: setup_offline_build.sh has been moved to scripts/setup/setup-offline-build.sh"
echo "Please update your references to use the new location"
echo "Executing from the new location..."
echo ""

exec "$(dirname "$0")/scripts/setup/setup-offline-build.sh"
