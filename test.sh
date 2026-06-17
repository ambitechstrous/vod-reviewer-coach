#!/bin/bash
set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <file_path>"
    exit 1
fi

FILE_PATH="${1//\\//}"
echo "Testing with file path: $FILE_PATH"

curl -X POST http://localhost:8080/analyze \
    -H "Content-Type: application/json" \
    -d "{\"file_path\": \"$FILE_PATH\", \"prompt\": \"Please analyze this gameplay video and tell me how this person did.\"}"