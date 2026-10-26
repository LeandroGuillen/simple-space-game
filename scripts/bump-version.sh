#!/usr/bin/env bash
set -euo pipefail

# Bump version script for Simple Space Game
# Updates all version locations from the single VERSION file

if [ $# -ne 1 ]; then
  echo "Usage: $0 <new-version>"
  echo "Example: $0 0.3.0"
  exit 1
fi

NEW_VERSION="$1"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Validate version format (simple check for X.Y.Z)
if ! echo "$NEW_VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "Error: Version must be in X.Y.Z format (e.g., 0.2.1)"
  exit 1
fi

echo "Bumping version to $NEW_VERSION"

# Update VERSION file
echo "$NEW_VERSION" > "$ROOT/VERSION"
echo "✓ Updated VERSION"

# Update server/version/VERSION
echo "$NEW_VERSION" > "$ROOT/server/version/VERSION"
echo "✓ Updated server/version/VERSION"

# Update package.json
cd "$ROOT/client-solid"
npm version "$NEW_VERSION" --no-git-tag-version --allow-same-version
echo "✓ Updated package.json and package-lock.json"

echo ""
echo "Version bumped to $NEW_VERSION in all locations."
echo "Next steps:"
echo "  1. Commit: git add -A && git commit -m 'bump version to $NEW_VERSION'"
echo "  2. Tag: git tag v$NEW_VERSION"
echo "  3. Push: git push origin master && git push origin v$NEW_VERSION"
