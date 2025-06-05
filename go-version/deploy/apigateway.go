package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
)

func (d *Deployer) createAPIGateway(ctx context.Context) error {
	// Create the API Gateway
	createAPIInput := &apigatewayv2.CreateApiInput{
		Name:         aws.String(d.apiName),
		ProtocolType: types.ProtocolTypeHttp,
		Description:  aws.String("Job scraper API Gateway"),
		Tags: map[string]string{
			"Project":     d.cfg.ProjectName,
			"Environment": d.cfg.Environment,
			"ManagedBy":   "aws-sdk-go",
		},
		CorsConfiguration: &types.Cors{
			AllowCredentials: aws.Bool(false),
			AllowHeaders:     []string{"content-type", "x-amz-date", "authorization", "x-api-key"},
			AllowMethods:     []string{"GET", "POST", "OPTIONS"},
			AllowOrigins:     []string{"*"},
			MaxAge:           aws.Int32(86400),
		},
	}

	apiResult, err := d.apiGwClient.CreateApi(ctx, createAPIInput)
	if err != nil {
		return fmt.Errorf("failed to create API Gateway: %w", err)
	}

	d.apiGatewayID = *apiResult.ApiId
	d.apiGatewayURL = *apiResult.ApiEndpoint
	fmt.Printf("   Created API Gateway: %s\n", d.apiGatewayID)
	fmt.Printf("   API Endpoint: %s\n", d.apiGatewayURL)

	return nil
}

func (d *Deployer) setupAPIIntegration(ctx context.Context) error {
	// Create Lambda integration
	integrationInput := &apigatewayv2.CreateIntegrationInput{
		ApiId:           aws.String(d.apiGatewayID),
		IntegrationType: types.IntegrationTypeAwsProxy,
		IntegrationUri:  aws.String(d.lambdaARN),
		PayloadFormatVersion: aws.String("2.0"),
		TimeoutInMillis: aws.Int32(29000), // 29 seconds (API Gateway max is 30s)
	}

	integrationResult, err := d.apiGwClient.CreateIntegration(ctx, integrationInput)
	if err != nil {
		return fmt.Errorf("failed to create API integration: %w", err)
	}

	integrationID := *integrationResult.IntegrationId
	fmt.Printf("   Created Lambda integration: %s\n", integrationID)

	// Create routes
	routes := []struct {
		routeKey string
		method   string
	}{
		{"POST /scrape", "POST"},
		{"GET /health", "GET"},
	}

	for _, route := range routes {
		routeInput := &apigatewayv2.CreateRouteInput{
			ApiId:    aws.String(d.apiGatewayID),
			RouteKey: aws.String(route.routeKey),
			Target:   aws.String(fmt.Sprintf("integrations/%s", integrationID)),
		}

		routeResult, err := d.apiGwClient.CreateRoute(ctx, routeInput)
		if err != nil {
			return fmt.Errorf("failed to create route %s: %w", route.routeKey, err)
		}

		fmt.Printf("   Created route: %s -> %s\n", route.routeKey, *routeResult.RouteId)
	}

	// Create deployment and stage
	deploymentInput := &apigatewayv2.CreateDeploymentInput{
		ApiId:       aws.String(d.apiGatewayID),
		Description: aws.String("Initial deployment"),
	}

	deploymentResult, err := d.apiGwClient.CreateDeployment(ctx, deploymentInput)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	fmt.Printf("   Created deployment: %s\n", *deploymentResult.DeploymentId)

	// Create stage
	stageInput := &apigatewayv2.CreateStageInput{
		ApiId:        aws.String(d.apiGatewayID),
		StageName:    aws.String("v1"),
		DeploymentId: deploymentResult.DeploymentId,
		Description:  aws.String("Version 1 stage"),
		Tags: map[string]string{
			"Project":     d.cfg.ProjectName,
			"Environment": d.cfg.Environment,
		},
	}

	stageResult, err := d.apiGwClient.CreateStage(ctx, stageInput)
	if err != nil {
		return fmt.Errorf("failed to create stage: %w", err)
	}

	fmt.Printf("   Created stage: %s\n", *stageResult.StageName)

	// Update the API Gateway URL to include the stage
	d.apiGatewayURL = fmt.Sprintf("%s/v1", d.apiGatewayURL)

	// Add permission for API Gateway to invoke Lambda
	err = d.addLambdaPermission(ctx, 
		"allow-api-gateway", 
		"apigateway.amazonaws.com",
		fmt.Sprintf("arn:aws:execute-api:%s:*:%s/*/*", d.cfg.Region, d.apiGatewayID),
	)
	if err != nil {
		return fmt.Errorf("failed to add API Gateway permission: %w", err)
	}

	return nil
}

func (d *Deployer) deleteAPIGateway(ctx context.Context) error {
	// List APIs to find our API
	listAPIsInput := &apigatewayv2.GetApisInput{}
	apis, err := d.apiGwClient.GetApis(ctx, listAPIsInput)
	if err != nil {
		return fmt.Errorf("failed to list APIs: %w", err)
	}

	// Find and delete our API
	for _, api := range apis.Items {
		if *api.Name == d.apiName {
			_, err := d.apiGwClient.DeleteApi(ctx, &apigatewayv2.DeleteApiInput{
				ApiId: api.ApiId,
			})
			if err != nil {
				return fmt.Errorf("failed to delete API Gateway: %w", err)
			}
			fmt.Printf("   Deleted API Gateway: %s\n", *api.ApiId)
			return nil
		}
	}

	fmt.Printf("   API Gateway %s not found (may already be deleted)\n", d.apiName)
	return nil
} 