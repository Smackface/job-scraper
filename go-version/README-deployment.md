# AWS Lambda Deployment Guide

## 🚀 Quick Deployment

### Prerequisites
- AWS CLI configured with appropriate permissions
- Go 1.23+ installed
- OpenAI API key

### Build & Deploy
```bash
# Windows
.\build.ps1

# Linux/Mac
./build.sh
```

This creates `lambda-deployment.zip` ready for AWS Lambda.

## 📦 Deployment Options

### Option 1: AWS Console (Manual)
1. Open AWS Lambda Console
2. Create new function → "Author from scratch"
3. Function name: `job-scraper-go`
4. Runtime: `Amazon Linux 2`
5. Architecture: `x86_64`
6. Upload `lambda-deployment.zip`

### Option 2: AWS CLI
```bash
# Create function
aws lambda create-function \
  --function-name job-scraper-go \
  --runtime provided.al2 \
  --role arn:aws:iam::YOUR_ACCOUNT:role/lambda-execution-role \
  --handler bootstrap \
  --zip-file fileb://lambda-deployment.zip

# Update function code
aws lambda update-function-code \
  --function-name job-scraper-go \
  --zip-file fileb://lambda-deployment.zip
```

## 🔧 Environment Variables

Set these in Lambda configuration:

```
OPENAI_KEY=your_openai_api_key_here
```

## 🌐 API Gateway Integration

### Request Format
```json
{
  "queryStringParameters": {
    "scraper": "hackernews"
  },
  "body": "[\"https://news.ycombinator.com/item?id=44159528\"]"
}
```

### Response Format
```json
{
  "statusCode": 200,
  "headers": {
    "Content-Type": "application/json",
    "Access-Control-Allow-Origin": "*"
  },
  "body": "{\"message\": \"Hacker News scraping completed successfully\", \"scraper\": \"hackernews\"}"
}
```

## ⚙️ Function Configuration

- **Memory**: 512 MB (minimum for OpenAI API calls)
- **Timeout**: 15 minutes (maximum for Lambda)
- **Concurrent executions**: 5-10 (to respect OpenAI rate limits)

## 🚨 Important Notes

1. **Rate Limiting**: Function includes built-in delays for OpenAI API
2. **File Output**: Creates logs in `/tmp/logs/` (ephemeral storage)
3. **LinkedIn**: Currently disabled pending ToS clarification
4. **Cold Starts**: First execution may take 10-30 seconds

## 🔐 IAM Permissions

Minimum required permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:*"
    }
  ]
}
``` 