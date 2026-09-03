# Build for Linux (required for Lambda)
GOOS=linux GOARCH=amd64 go build -o bootstrap ./cmd/analyzer

# Package. Remove executable once packaged, since Lambda doesn't need it.
zip deployment.zip bootstrap
rm bootstrap

# Deploy (update existing function)
aws lambda update-function-code \
  --function-name VodAnalyzer \
  --zip-file fileb://deployment.zip

# Clean up zip file after deployment
rm deployment.zip
