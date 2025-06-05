package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"

)

func (d *Deployer) getAccountInfo(ctx context.Context) error {
	result, err := d.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return err
	}
	
	d.accountID = *result.Account
	fmt.Printf("   Account ID: %s\n", d.accountID)
	fmt.Printf("   Region: %s\n", d.cfg.Region)
	return nil
}

func (d *Deployer) readZipFile() ([]byte, error) {
	return os.ReadFile(d.cfg.LambdaZipPath)
}

func (d *Deployer) createLambdaFunction(ctx context.Context) error {
	// Check if function already exists
	_, err := d.lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(d.functionName),
	})
	if err == nil {
		return fmt.Errorf("Lambda function %s already exists", d.functionName)
	}

	// Read the deployment zip
	zipData, err := d.readZipFile()
	if err != nil {
		return fmt.Errorf("failed to read zip file: %w", err)
	}

	// Create the Lambda function
	createInput := &lambda.CreateFunctionInput{
		FunctionName: aws.String(d.functionName),
		Runtime:      types.RuntimeProvidedal2,
		Role:         aws.String(d.roleARN),
		Handler:      aws.String("bootstrap"),
		Code: &types.FunctionCode{
			ZipFile: zipData,
		},
		Description: aws.String("Job scraper Lambda function"),
		MemorySize:  aws.Int32(d.cfg.LambdaMemorySize),
		Timeout:     aws.Int32(d.cfg.LambdaTimeout),
		Environment: &types.Environment{
			Variables: map[string]string{
				"OPENAI_KEY": d.cfg.OpenAIAPIKey,
			},
		},
		Tags: map[string]string{
			"Project":     d.cfg.ProjectName,
			"Environment": d.cfg.Environment,
			"ManagedBy":   "aws-sdk-go",
		},
	}

	result, err := d.lambdaClient.CreateFunction(ctx, createInput)
	if err != nil {
		return fmt.Errorf("failed to create Lambda function: %w", err)
	}

	d.lambdaARN = *result.FunctionArn
	fmt.Printf("   Created Lambda function: %s\n", d.lambdaARN)

	// Wait for function to be active
	return d.waitForLambdaActive(ctx)
}

func (d *Deployer) waitForLambdaActive(ctx context.Context) error {
	fmt.Print("   Waiting for Lambda function to be active")
	
	waiter := lambda.NewFunctionActiveWaiter(d.lambdaClient)
	maxWaitTime := 5 * time.Minute
	
	err := waiter.Wait(ctx, &lambda.GetFunctionConfigurationInput{
		FunctionName: aws.String(d.functionName),
	}, maxWaitTime)
	
	if err != nil {
		return fmt.Errorf("Lambda function did not become active: %w", err)
	}
	
	fmt.Println(" ✅")
	return nil
}

func (d *Deployer) deleteLambdaFunction(ctx context.Context) error {
	_, err := d.lambdaClient.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: aws.String(d.functionName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete Lambda function: %w", err)
	}
	
	fmt.Printf("   Deleted Lambda function: %s\n", d.functionName)
	return nil
}

func (d *Deployer) addLambdaPermission(ctx context.Context, statementId, principal, sourceArn string) error {
	_, err := d.lambdaClient.AddPermission(ctx, &lambda.AddPermissionInput{
		FunctionName: aws.String(d.functionName),
		StatementId:  aws.String(statementId),
		Action:       aws.String("lambda:InvokeFunction"),
		Principal:    aws.String(principal),
		SourceArn:    aws.String(sourceArn),
	})
	
	if err != nil {
		return fmt.Errorf("failed to add Lambda permission: %w", err)
	}
	
	fmt.Printf("   Added permission for %s to invoke Lambda\n", principal)
	return nil
} 