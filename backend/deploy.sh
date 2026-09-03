# Build for Linux (required for Lambda)
GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

# Package. Remove executable once packaged, since Lambda doesn't need it.
zip deployment.zip server
rm server

# Deploy (update existing function)
aws lambda update-function-code \
  --function-name VodAnalyzer \
  --zip-file fileb://deployment.zip

# Clean up zip file after deployment
rm deployment.zip
