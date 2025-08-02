#!/bin/bash
# Calculate lines of code excluding comments and empty lines and test files
# Get https://github.com/AlDanial/cloc#install-via-package-manager
echo "Go files"
git ls-files | grep -v '_test.go' | grep '\.go$' | xargs cloc
echo "Test files"
git ls-files | grep '_test.go' | xargs cloc
