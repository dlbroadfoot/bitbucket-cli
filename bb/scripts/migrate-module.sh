#!/bin/bash
# migrate-module.sh - Migrate module path from github.com/cli/bb/v2 to github.com/dlbroadfoot/bitbucket-cli
#
# Usage: ./scripts/migrate-module.sh
#
# This script updates all Go import paths from the old module path to the new one.
# Run this BEFORE pushing to the new repository.

set -e

OLD_MODULE="github.com/cli/bb/v2"
NEW_MODULE="github.com/dlbroadfoot/bitbucket-cli"

echo "Migrating module path..."
echo "  Old: $OLD_MODULE"
echo "  New: $NEW_MODULE"
echo ""

# Check if we're in the right directory
if [[ ! -f "go.mod" ]]; then
    echo "Error: go.mod not found. Please run this script from the project root."
    exit 1
fi

# Count occurrences
COUNT=$(grep -r "$OLD_MODULE" --include="*.go" --include="go.mod" . 2>/dev/null | wc -l | tr -d ' ')
echo "Found $COUNT occurrences to replace"
echo ""

# Perform the replacement
echo "Updating go.mod..."
sed -i.bak "s|$OLD_MODULE|$NEW_MODULE|g" go.mod
rm -f go.mod.bak

echo "Updating Go files..."
find . -name "*.go" -type f | while read -r file; do
    if grep -q "$OLD_MODULE" "$file" 2>/dev/null; then
        sed -i.bak "s|$OLD_MODULE|$NEW_MODULE|g" "$file"
        rm -f "${file}.bak"
        echo "  Updated: $file"
    fi
done

echo ""
echo "Migration complete!"
echo ""
echo "Next steps:"
echo "  1. Run 'go mod tidy' to update dependencies"
echo "  2. Run 'go build ./...' to verify the build"
echo "  3. Run 'go test ./...' to verify tests"
echo "  4. Commit the changes"
