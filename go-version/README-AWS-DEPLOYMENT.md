# 🚀 AWS Job Scraper - Complete Deployment Guide

This project provides a **complete serverless job scraper solution** built with Go and deployed using **AWS SDK for Go v2** for Infrastructure as Code.

## 🎯 Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   API Gateway   │────│  Lambda Function │────│   OpenAI API    │
│   (HTTP API)    │    │    (Go 1.23)    │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       
         │                       │               
    ┌─────────┐           ┌─────────────────┐   
    │  CORS   │           │  CloudWatch     │   
    │ Enabled │           │     Logs        │   
    └─────────┘           └─────────────────┘   
```

## 📁 Project Structure

```
go-version/
├── main.go                    # Lambda function code
├── go.mod                     # Lambda dependencies
├── internal/                  # Scraper implementations
│   ├── hackernewsscraper.go  # Hacker News scraper
│   └── linkedinscraper.go    # LinkedIn scraper (disabled)
├── build.ps1                 # Windows build script
├── build.sh                  # Linux/Mac build script
├── lambda-deployment.zip     # Generated deployment package
├── deploy/                   # AWS SDK deployment tool
│   ├── main.go              # CLI interface
│   ├── deployer.go          # Main deployment logic
│   ├── lambda.go            # Lambda management
│   ├── iam.go               # IAM role management  
│   ├── apigateway.go        # API Gateway setup
│   ├── go.mod               # Deployment tool dependencies
│   └── README.md            # Deployment tool docs
└── README-AWS-DEPLOYMENT.md  # This file
```

## 🚀 Quick Start (5-Minute Deployment)

### Prerequisites
✅ AWS CLI configured with admin permissions  
✅ Go 1.23+ installed  
✅ OpenAI API key  

### Step 1: Build Lambda Function
```bash
# Build the Lambda deployment package
.\build.ps1  # Windows
# or
./build.sh   # Linux/Mac
```

### Step 2: Deploy Infrastructure
```bash
# Navigate to deployment tool
cd deploy

# Deploy everything to AWS
go run . -deploy -openai-key="sk-proj-your-key-here"
```

### Step 3: Test Your API
```bash
# Test the deployed API (use the URL from deployment output)
curl -X POST \
  "https://your-api-id.execute-api.us-east-1.amazonaws.com/v1/scrape?scraper=hackernews" \
  -H "Content-Type: application/json" \
  -d '["https://news.ycombinator.com/item?id=44159528"]'
```

## 🛠️ What Gets Deployed

### **AWS Lambda Function**
- **Name**: `job-scraper-dev`
- **Runtime**: Go 1.23 on Amazon Linux 2
- **Memory**: 512 MB
- **Timeout**: 15 minutes
- **Environment**: `OPENAI_KEY` securely stored

### **API Gateway (HTTP API)**
- **Type**: HTTP API (cost-optimized)
- **CORS**: Enabled for web frontends
- **Routes**:
  - `POST /scrape?scraper=hackernews` → Job scraping
  - `GET /health` → Health check
- **Stage**: `v1`

### **IAM Role & Policies**
- **Role**: `job-scraper-lambda-role-dev`
- **Permissions**: CloudWatch Logs only (principle of least privilege)
- **Policy**: Custom execution policy

## 📊 Features & Capabilities

### **Current Features**
✅ **Hacker News Scraping** - Extracts job postings with specific tech stacks  
✅ **OpenAI Integration** - Intelligent content analysis and filtering  
✅ **Rate Limiting** - Built-in delays to respect API limits  
✅ **Error Handling** - Comprehensive error handling and retry logic  
✅ **CORS Support** - Ready for web frontend integration  
✅ **Infrastructure as Code** - Fully automated AWS deployment  

### **Built-in Intelligence**
- 🎯 **Tech Stack Filtering**: Focuses on React, Vue, Go, AWS technologies
- 🚫 **Smart Exclusions**: Filters out C/C++ and irrelevant posts  
- 📝 **Content Analysis**: Extracts company info, contact details, job requirements
- 🔄 **Retry Logic**: Handles rate limits and API failures gracefully

## 🔧 Development Workflow

### **Making Changes**
```bash
# 1. Edit Lambda function code
code main.go  # or internal/*.go

# 2. Rebuild deployment package  
.\build.ps1

# 3. Update Lambda code only (fast)
cd deploy
go run . -update
```

### **Full Redeployment**
```bash
# Complete infrastructure teardown and rebuild
cd deploy
go run . -destroy  # Remove everything
go run . -deploy   # Deploy fresh
```

## 🎯 Future Roadmap

### **Phase 1: Current** ✅
- [x] Lambda + API Gateway deployment
- [x] Hacker News scraping
- [x] OpenAI integration
- [x] Infrastructure as Code

### **Phase 2: User Management** 🚧  
- [ ] **DynamoDB Tables** - User profiles and API keys
- [ ] **Cognito Integration** - User authentication
- [ ] **User-specific API Keys** - Replace environment variables
- [ ] **Usage Tracking** - Monitor scraping quotas

### **Phase 3: Enhanced Features** 📋
- [ ] **Multiple Job Sources** - Indeed, LinkedIn (when ToS allows), Dice, more planned
- [ ] **Real-time Notifications** - WebSocket/SNS integration  
- [ ] **Job Filtering API** - Advanced search capabilities
- [ ] **Data Persistence** - Store results in DynamoDB
- [ ] **Frontend Dashboard** - React/Vue web interface

### **Phase 4: Enterprise** 🚀
- [ ] **Multi-tenant Architecture** - Support multiple organizations
- [ ] **Analytics Dashboard** - Job market insights
- [ ] **Webhook Integration** - Third-party notifications
- [ ] **API Rate Limiting** - Per-user quotas
- [ ] **Caching Layer** - Redis/ElastiCache integration

## 💰 Cost Estimate

**Current Architecture (Light Usage):**
- **Lambda**: ~$0.20/month (1M requests)
- **API Gateway**: ~$3.50/month (1M requests)  
- **CloudWatch Logs**: ~$0.50/month
- **Total**: **~$4.20/month**

**With Full Roadmap:**
- **DynamoDB**: +$2.50/month
- **Cognito**: +$0.00 (50,000 free MAU)
- **S3**: +$0.25/month  
- **Estimated Total**: **~$7.00/month**

## 🛡️ Security Features

- ✅ **IAM Least Privilege** - Minimal required permissions only
- ✅ **Environment Variables** - Secure API key storage
- ✅ **HTTPS Only** - All API communication encrypted
- ✅ **CORS Configuration** - Controlled web access
- ✅ **Resource Tagging** - Complete resource management
- ✅ **No Hardcoded Secrets** - All sensitive data externalized

## 🐛 Troubleshooting

### **Common Issues**

**"Lambda function already exists"**
```bash
cd deploy
go run . -destroy && go run . -deploy
```

**"Zip file not found"**
```bash
.\build.ps1  # Rebuild deployment package
```

**"Permission denied"**
```bash
aws sts get-caller-identity  # Verify AWS access
```

### **Debug Mode**
```bash
# Enable verbose AWS SDK logging
export AWS_SDK_LOAD_CONFIG=1
export AWS_LOG_LEVEL=DEBUG
cd deploy && go run . -deploy
```

## 📞 Support & Contributing

### **Getting Help**
1. Check the [Deployment Tool README](deploy/README.md)
2. Review CloudWatch Logs in AWS Console
3. Test with minimal example requests

### **Contributing**
1. Fork the repository
2. Create feature branch
3. Add comprehensive tests
4. Update documentation
5. Submit pull request

## 🏆 Why This Architecture?

### **Benefits of AWS SDK for Go**
- ✅ **Type Safety** - Compile-time error checking
- ✅ **Native Performance** - No external dependencies
- ✅ **Maintainability** - Same language as Lambda function
- ✅ **AWS Integration** - First-class AWS service support
- ✅ **Cost Effective** - No third-party IaC tools required

### **Scalability Considerations**
- 🔄 **Stateless Design** - Horizontal scaling ready
- 📊 **Serverless Architecture** - Auto-scaling built-in
- 🗄️ **Database Ready** - DynamoDB integration prepared
- 🌐 **CDN Ready** - CloudFront integration possible
- 🔐 **Auth Ready** - Cognito integration structured

---

## 🎉 You're Ready!

Your job scraper is now deployed and ready to scale. Start with the basic functionality and gradually add features from the roadmap as your needs grow.

**Next Steps:**
1. Test your deployed API
2. Monitor CloudWatch logs  
3. Plan your first enhancement
4. Consider adding user management
5. Build your frontend interface

Happy scraping! 🕷️✨ 