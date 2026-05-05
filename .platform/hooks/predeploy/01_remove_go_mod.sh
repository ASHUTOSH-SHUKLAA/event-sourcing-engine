#!/bin/bash
# Remove go.mod to prevent EB from trying to build from source
# We're using a pre-built binary instead
rm -f /var/app/staging/go.mod /var/app/staging/go.sum
echo "Removed go.mod and go.sum to use pre-built binary"
