#!/bin/bash

cleanup() {
  # Stop the web server if it is still running
  kill "$PID"

  # Clean up the binary
  rm -rf ./bin
}

# Build the Go binary
go build -o ./bin/api ./cmd/api

# Start the web server in the background
./bin/api &

# Capture the PID of the server process so we can kill it later
PID=$!

# Call the cleanup function on exit
trap cleanup EXIT ERR

# Wait for the server to start
sleep 2

# Make a GET request to the server and check the response
RESPONSE=$(curl http://localhost:4000/v1/healthcheck)

# Check if the response contains the expected string
EXPECTED='{"status":"available","system_info":{"environment":"development","version":"1.0.0"}}'
if [[ $RESPONSE != "$EXPECTED" ]]; then
  echo "Smoke test failed. Server did not respond as expected."
  echo "Expected: $EXPECTED"
  echo "Actual  : $RESPONSE"
  exit 1
fi

echo "=====SMOKE TEST PASSED====="
exit 0
