#!/bin/sh
#
# Runs all tests.
echo "Running all Go tests.." >&2
goapp test ./...
