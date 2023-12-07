#!/bin/bash

cleanup() {
  # Stop the web server if it is still running
  echo "Stopping server with PID: $serverPid"
  kill "$serverPid"

  # Clean up the binary
  rm -rf ./bin
}


setup() {
  # Build the Go binary
  go build -o ./bin/server ./cmd/api

  # Start the web server in the background and discard all output
  ./bin/server -limiter-enabled=false > /dev/null &

  # Capture the PID of the server process so we can kill it later
  serverPid=$!
  echo "Server PID: $serverPid"

  # wait for the server to start
  sleep 2

  # Register a new user
  response=$(curl -X POST --location "http://localhost:4000/v1/users" \
    -H "Content-Type: application/json" \
    -d '{"name":"Alice", "email":"alice@example.com", "password":"pa55word"}' -D -)

  # check that the response code is 202 Accepted
  status=$(echo "$response" | head -n 1 | cut -d$' ' -f2)
  if [[ "$status" -ne 202 ]]; then
    echo "Load test failed. Unexpected response code."
    echo "Expected: 202"
    echo "Actual  : $status"
    exit 1
  fi
}

runLoadTest() {
  echo "=====LOAD TEST STARTED====="
  hey -d '{"email": "alice@example.com", "password": "pa55word"}' \
     -m "POST" http://localhost:4000/v1/tokens/authentication
  echo "=====LOAD TEST COMPLETED====="
}

# Register cleanup function on exit or error
trap cleanup EXIT ERR

# Create global variable called PID
export serverPid=0

# Call the setup function
setup

# start load test
runLoadTest

while true; do
    echo "Choose an option:"
    echo "1. Exit the script"
    echo "2. Re-run the load test"
    read -r -p "Enter your choice (1 or 2): " choice
    case $choice in
        1)
            echo "Stopping the server and exiting the script"
            exit 0
            ;;
        2)
            echo "Re-running load test"
            runLoadTest
            ;;
        *)
            echo "Invalid choice, please choose 1 or 2."
            ;;
    esac
done
