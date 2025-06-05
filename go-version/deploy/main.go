package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"

)

const (
	// Project configuration
	ProjectName = "job-scraper"
	Environment = "dev"
	Region      = "us-east-1"
)

type DeploymentConfig struct {
	ProjectName       string
	Environment       string
	Region           string
	OpenAIAPIKey     string
	LambdaZipPath    string
	LambdaMemorySize int32
	LambdaTimeout    int32
}

func main() {
	log.Println("🚀 Starting AWS Job Scraper Deployment...")

	// Parse command line flags
	var (
		deploy   = flag.Bool("deploy", false, "Deploy the infrastructure")
		destroy  = flag.Bool("destroy", false, "Destroy the infrastructure")
		update   = flag.Bool("update", false, "Update Lambda function code only")
		openaiKey = flag.String("openai-key", "", "OpenAI API Key (or set OPENAI_KEY env var)")
		zipPath   = flag.String("zip", "../lambda-deployment.zip", "Path to Lambda deployment zip")
	)
	flag.Parse()

	// Validate arguments
	if !*deploy && !*destroy && !*update {
		fmt.Println("Usage:")
		fmt.Println("  deploy -deploy -openai-key=sk-...")
		fmt.Println("  deploy -update")
		fmt.Println("  deploy -destroy")
		os.Exit(1)
	}

	// Get OpenAI API key
	apiKey := *openaiKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_KEY")
	}
	if apiKey == "" && (*deploy || *update) {
		log.Fatal("❌ OpenAI API key is required. Use -openai-key flag or set OPENAI_KEY environment variable")
	}

	// Create deployment configuration
	cfg := &DeploymentConfig{
		ProjectName:       ProjectName,
		Environment:       Environment,
		Region:           Region,
		OpenAIAPIKey:     apiKey,
		LambdaZipPath:    *zipPath,
		LambdaMemorySize: 512,
		LambdaTimeout:    900, // 15 minutes
	}

	// Load AWS configuration
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		log.Fatalf("❌ Failed to load AWS config: %v", err)
	}

	// Create deployer
	deployer := NewDeployer(awsCfg, cfg)

	// Execute requested action
	switch {
	case *deploy:
		log.Println("📦 Deploying full infrastructure...")
		if err := deployer.Deploy(ctx); err != nil {
			log.Fatalf("❌ Deployment failed: %v", err)
		}
		log.Println("✅ Deployment completed successfully!")

	case *update:
		log.Println("🔄 Updating Lambda function code...")
		if err := deployer.UpdateLambda(ctx); err != nil {
			log.Fatalf("❌ Update failed: %v", err)
		}
		log.Println("✅ Lambda function updated successfully!")

	case *destroy:
		log.Println("💥 Destroying infrastructure...")
		if err := deployer.Destroy(ctx); err != nil {
			log.Fatalf("❌ Destroy failed: %v", err)
		}
		log.Println("✅ Infrastructure destroyed successfully!")
	}
} 