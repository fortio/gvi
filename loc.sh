#!/bin/bash
# Calculate lines of code excluding comments and empty lines and test files
# go install github.com/ldemailly/go-scratch/gloc@latest
echo "Go files"
git ls-files | grep -v '_test.go' | grep '\.go$' | xargs gloc
echo "Test files"
git ls-files | grep '_test.go' | xargs gloc
