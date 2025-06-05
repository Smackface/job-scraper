package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type Deployer struct {
	cfg           *DeploymentConfig
	awsCfg        aws.Config
	lambdaClient  *lambda.Client
	iamClient     *iam.Client
	apiGwClient   *apigatewayv2.Client
	stsClient     *sts.Client
	
	// Resource names
	functionName     string
	roleName         string
	apiName          string
	
	// AWS account info
	accountID        string
	
	// Deployment state
	lambdaARN        string
	roleARN          string
	apiGatewayID     string
	apiGatewayURL    string
}

func NewDeployer(awsCfg aws.Config, cfg *DeploymentConfig) *Deployer {
	return &Deployer{
		cfg:          cfg,
		awsCfg:       awsCfg,
		lambdaClient: lambda.NewFromConfig(awsCfg),
		iamClient:    iam.NewFromConfig(awsCfg),
		apiGwClient:  apigatewayv2.NewFromConfig(awsCfg),
		stsClient:    sts.NewFromConfig(awsCfg),
		
		functionName: fmt.Sprintf("%s-%s", cfg.ProjectName, cfg.Environment),
		roleName:     fmt.Sprintf("%s-lambda-role-%s", cfg.ProjectName, cfg.Environment),
		apiName:      fmt.Sprintf("%s-api-%s", cfg.ProjectName, cfg.Environment),
	}
}

// Deploy creates the full infrastructure
func (d *Deployer) Deploy(ctx context.Context) error {
	log.Println("🔑 Getting account information...")
	if err := d.getAccountInfo(ctx); err != nil {
		return fmt.Errorf("failed to get account info: %w", err)
	}

	log.Println("🛡️  Creating IAM role...")
	if err := d.createIAMRole(ctx); err != nil {
		return fmt.Errorf("failed to create IAM role: %w", err)
	}

	log.Println("🚀 Creating Lambda function...")
	if err := d.createLambdaFunction(ctx); err != nil {
		return fmt.Errorf("failed to create Lambda function: %w", err)
	}

	log.Println("🌐 Creating API Gateway...")
	if err := d.createAPIGateway(ctx); err != nil {
		return fmt.Errorf("failed to create API Gateway: %w", err)
	}

	log.Println("🔗 Setting up API Gateway Lambda integration...")
	if err := d.setupAPIIntegration(ctx); err != nil {
		return fmt.Errorf("failed to setup API integration: %w", err)
	}

	d.printDeploymentInfo()
	return nil
}

// UpdateLambda updates only the Lambda function code
func (d *Deployer) UpdateLambda(ctx context.Context) error {
	log.Printf("📝 Reading Lambda zip file: %s", d.cfg.LambdaZipPath)
	zipData, err := d.readZipFile()
	if err != nil {
		return fmt.Errorf("failed to read zip file: %w", err)
	}

	log.Println("🔄 Updating Lambda function code...")
	_, err = d.lambdaClient.UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(d.functionName),
		ZipFile:      zipData,
	})
	if err != nil {
		return fmt.Errorf("failed to update Lambda function: %w", err)
	}

	return nil
}

// Destroy removes all created resources
func (d *Deployer) Destroy(ctx context.Context) error {
	log.Println("🗑️  Deleting API Gateway...")
	if err := d.deleteAPIGateway(ctx); err != nil {
		log.Printf("⚠️  Warning: failed to delete API Gateway: %v", err)
	}

	log.Println("🗑️  Deleting Lambda function...")
	if err := d.deleteLambdaFunction(ctx); err != nil {
		log.Printf("⚠️  Warning: failed to delete Lambda function: %v", err)
	}

	log.Println("🗑️  Deleting IAM role...")
	if err := d.deleteIAMRole(ctx); err != nil {
		log.Printf("⚠️  Warning: failed to delete IAM role: %v", err)
	}

	return nil
}

func (d *Deployer) printDeploymentInfo() {
	log.Println("\n🎉 Deployment Summary:")
	log.Printf("   Lambda Function: %s", d.functionName)
	log.Printf("   Lambda ARN: %s", d.lambdaARN)
	log.Printf("   IAM Role: %s", d.roleName)
	log.Printf("   API Gateway: %s", d.apiName)
	if d.apiGatewayURL != "" {
		log.Printf("   API Endpoint: %s", d.apiGatewayURL)
	}
	log.Println("\n📖 Usage:")
	log.Println("   curl -X POST \\")
	log.Printf("     \"%s/scrape?scraper=hackernews\" \\\n", d.apiGatewayURL)
	log.Println("     -H \"Content-Type: application/json\" \\")
	log.Println("     -d '[\"https://news.ycombinator.com/item?id=44159528\"]'")
} 