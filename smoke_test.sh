#!/bin/bash

cleanup() {
  # Stop the web server if it is still running
  kill "$PID"

  # Clean up the binary
  rm -rf ./bin
}

# Build the Go binary
go build -o ./bin/server ./cmd/api

# Start the web server in the background
./bin/server &

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

# make a POST request to localhost:4000/v1/movies with a JSON request body
curl \
  --silent \
  --header 'Content-Type: application/json' \
  --request POST \
  --data '{"title":"Black Panther","year":2018,"runtime":"134 mins","genres":["sci-fi", "action", "adventure"]}' \
  http://localhost:4000/v1/movies


# make a GET request to localhost:4000/v1/movies/1 and check the response
RESPONSE=$(curl http://localhost:4000/v1/movies/1)

# Check if the response contains the expected string
EXPECTED='{"movie":{"id":1,"title":"Black Panther","year":2018,"runtime":"134 mins","genres":["sci-fi","action","adventure"],"version":1}}'
if [[ $RESPONSE != "$EXPECTED" ]]; then
  echo "Smoke test failed. Server did not respond as expected."
  echo "Expected: $EXPECTED"
  echo "Actual  : $RESPONSE"
  exit 1
fi

echo "=====SMOKE TEST PASSED====="
exit 0
