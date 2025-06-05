# AWS Job Scraper Deployment Tool

This tool uses the **AWS SDK for Go v2** to programmatically deploy and manage your job scraper infrastructure.

## 🚀 Features

- **Full Infrastructure Deployment**: Lambda + API Gateway + IAM roles
- **Incremental Updates**: Update just Lambda code without touching infrastructure  
- **Clean Teardown**: Remove all resources when done
- **Type-Safe**: Written in Go with full type safety
- **Future-Ready**: Structured for easy DynamoDB and Cognito additions

## 📋 Prerequisites

1. **AWS CLI configured** with appropriate permissions
2. **Go 1.23+** installed
3. **OpenAI API key**
4. **Lambda deployment package** built (`../lambda-deployment.zip`)

## 🔧 Setup

```bash
# Navigate to deploy directory
cd deploy

# Install dependencies
go mod tidy
```

## 🚀 Usage

### 1. Initial Deployment
```bash
# Deploy everything
go run . -deploy -openai-key="sk-proj-your-key-here"

# Or use environment variable
export OPENAI_KEY="sk-proj-your-key-here"
go run . -deploy
```

### 2. Update Lambda Code Only
```bash
# After making changes to your Lambda function
cd ..
./build.ps1  # Rebuild the deployment package
cd deploy
go run . -update
```

### 3. Destroy Infrastructure
```bash
go run . -destroy
```

## 📁 Project Structure

```
deploy/
├── main.go          # CLI interface and orchestration
├── deployer.go      # Main deployment logic
├── lambda.go        # Lambda function management
├── iam.go          # IAM role and policy management
├── apigateway.go   # API Gateway setup
├── go.mod          # Dependencies
└── README.md       # This file
```

## 🌐 What Gets Created

### **Lambda Function**
- Function name: `job-scraper-dev`
- Runtime: `provided.al2` (custom Go runtime)
- Memory: 512 MB
- Timeout: 15 minutes
- Environment variables: `OPENAI_KEY`

### **IAM Role**  
- Role name: `job-scraper-lambda-role-dev`
- Permissions: CloudWatch Logs access
- Trust policy: Lambda service

### **API Gateway**
- Type: HTTP API (cheaper than REST API)
- CORS enabled for web frontend
- Routes:
  - `POST /scrape` → Lambda function
  - `GET /health` → Lambda function
- Stage: `v1`

## 🔑 API Usage

Once deployed, you can call your API:

```bash
# Test the scraper
curl -X POST \
  "https://your-api-id.execute-api.us-east-1.amazonaws.com/v1/scrape?scraper=hackernews" \
  -H "Content-Type: application/json" \
  -d '["https://news.ycombinator.com/item?id=44159528"]'

# Health check
curl "https://your-api-id.execute-api.us-east-1.amazonaws.com/v1/health"
```

## 🚧 Future Expansion

The deployer is structured to easily add:

### **DynamoDB Tables**
```go
// Add to deployer.go
func (d *Deployer) createDynamoDB(ctx context.Context) error {
    // DynamoDB table creation logic
}
```

### **Cognito User Pool**
```go
// Add to deployer.go  
func (d *Deployer) createCognito(ctx context.Context) error {
    // Cognito setup logic
}
```

### **S3 Buckets**
```go
// For storing scraper results
func (d *Deployer) createS3Bucket(ctx context.Context) error {
    // S3 bucket creation
}
```

## 🛡️ Security Best Practices

- **IAM Least Privilege**: Only grants necessary permissions
- **Environment Variables**: Sensitive data stored securely
- **Resource Tagging**: All resources tagged for management
- **CORS Configuration**: Restricted to necessary headers/methods

## 🔧 Customization

Edit constants in `main.go`:
```go
const (
    ProjectName = "job-scraper"    // Change project name
    Environment = "dev"            // Change environment
    Region      = "us-east-1"     // Change AWS region
)
```

## 🐛 Troubleshooting

### **"Lambda function already exists"**
Use `-update` flag instead of `-deploy`, or `-destroy` then `-deploy`.

### **"IAM role not found"**  
IAM has eventual consistency. Wait 30 seconds and try again.

### **"Permission denied"**
Ensure your AWS CLI has sufficient permissions:
- `lambda:*`
- `iam:*`  
- `apigateway:*`
- `sts:GetCallerIdentity`

### **"Zip file not found"**
Build the Lambda deployment package first:
```bash
cd ..
./build.ps1
cd deploy
```

## 📊 Cost Estimation

**Monthly costs for light usage:**
- Lambda: ~$0.20 (1M requests)
- API Gateway: ~$3.50 (1M requests)  
- CloudWatch Logs: ~$0.50
- **Total: ~$4.20/month**

## 🎯 Next Steps

1. **Deploy your infrastructure**: `go run . -deploy`
2. **Test the API endpoints**
3. **Set up monitoring** in CloudWatch
4. **Add DynamoDB** for user management
5. **Implement Cognito** for authentication
6. **Build your frontend** using the API 