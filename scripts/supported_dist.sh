#!/bin/bash

set -euo pipefail # Exit on error, undefined vars, and pipe failures

# Get all supported platforms from go tool
platforms=$(go tool dist list)

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Arrays to store successful builds
declare -a cgo_enabled_platforms
declare -a cgo_disabled_platforms

echo "Testing builds for all platforms..."
echo "-----------------------------------"

# Function to attempt build and record result
try_build() {
    local platform=$1
    local cgo_setting=$2
    local os=${platform%/*}
    local arch=${platform#*/}

    echo -n "Building for $platform (CGO ${cgo_setting})... "
    if CGO_ENABLED=$cgo_setting GOOS=$os GOARCH=$arch go build -o /dev/null 2>/dev/null; then
        echo -e "${GREEN}✅ Success${NC}"
        return 0
    else
        echo -e "${RED}❌ Build failed${NC}"
        return 1
    fi
}

# Check each platform with and without CGO
while IFS= read -r platform; do
    # Try CGO enabled build
    if try_build "$platform" 1; then
        cgo_enabled_platforms+=("$platform")
    fi

    # Try CGO disabled build
    if try_build "$platform" 0; then
        cgo_disabled_platforms+=("$platform")
    fi

    echo "-----------------------------------"
done <<<"$platforms"

echo "Build test complete"

# Print summary
echo
echo "Summary of Supported Platforms:"
echo "------------------------------"
echo "Platforms that support CGO enabled builds (${#cgo_enabled_platforms[@]} total):"
printf '%s\n' "${cgo_enabled_platforms[@]}" | sort
echo
echo "Platforms that support CGO disabled builds (${#cgo_disabled_platforms[@]} total):"
printf '%s\n' "${cgo_disabled_platforms[@]}" | sort
