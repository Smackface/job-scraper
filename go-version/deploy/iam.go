package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

)

const (
	lambdaAssumeRolePolicy = `{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Principal": {
				"Service": "lambda.amazonaws.com"
			},
			"Action": "sts:AssumeRole"
		}
	]
}`

	lambdaExecutionPolicy = `{
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
}`
)

func (d *Deployer) createIAMRole(ctx context.Context) error {
	// Check if role already exists
	getResult, err := d.iamClient.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(d.roleName),
	})
	if err == nil {
		d.roleARN = *getResult.Role.Arn
		fmt.Printf("   Using existing IAM role: %s\n", d.roleARN)
		return nil
	}

	// Create the execution role
	createRoleInput := &iam.CreateRoleInput{
		RoleName:                 aws.String(d.roleName),
		AssumeRolePolicyDocument: aws.String(lambdaAssumeRolePolicy),
		Description:              aws.String("Lambda execution role for job scraper"),
		Tags: []types.Tag{
			{
				Key:   aws.String("Project"),
				Value: aws.String(d.cfg.ProjectName),
			},
			{
				Key:   aws.String("Environment"),
				Value: aws.String(d.cfg.Environment),
			},
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("aws-sdk-go"),
			},
		},
	}

	createResult, err := d.iamClient.CreateRole(ctx, createRoleInput)
	if err != nil {
		return fmt.Errorf("failed to create IAM role: %w", err)
	}

	d.roleARN = *createResult.Role.Arn
	fmt.Printf("   Created IAM role: %s\n", d.roleARN)

	// Create and attach custom policy for Lambda execution
	policyName := fmt.Sprintf("%s-execution-policy", d.roleName)
	
	// Create the policy
	createPolicyInput := &iam.CreatePolicyInput{
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(lambdaExecutionPolicy),
		Description:    aws.String("Lambda execution policy for job scraper"),
		Tags: []types.Tag{
			{
				Key:   aws.String("Project"),
				Value: aws.String(d.cfg.ProjectName),
			},
			{
				Key:   aws.String("Environment"),
				Value: aws.String(d.cfg.Environment),
			},
		},
	}

	policyResult, err := d.iamClient.CreatePolicy(ctx, createPolicyInput)
	if err != nil {
		return fmt.Errorf("failed to create IAM policy: %w", err)
	}

	// Attach the policy to the role
	attachPolicyInput := &iam.AttachRolePolicyInput{
		RoleName:  aws.String(d.roleName),
		PolicyArn: policyResult.Policy.Arn,
	}

	_, err = d.iamClient.AttachRolePolicy(ctx, attachPolicyInput)
	if err != nil {
		return fmt.Errorf("failed to attach policy to role: %w", err)
	}

	fmt.Printf("   Attached execution policy: %s\n", *policyResult.Policy.Arn)

	// Wait for role to be available (IAM eventual consistency)
	fmt.Print("   Waiting for IAM role to be available")
	time.Sleep(10 * time.Second)
	fmt.Println(" ✅")

	return nil
}

func (d *Deployer) deleteIAMRole(ctx context.Context) error {
	// First, detach all policies
	listPoliciesInput := &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(d.roleName),
	}

	policies, err := d.iamClient.ListAttachedRolePolicies(ctx, listPoliciesInput)
	if err == nil {
		for _, policy := range policies.AttachedPolicies {
			// Detach the policy
			_, err := d.iamClient.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
				RoleName:  aws.String(d.roleName),
				PolicyArn: policy.PolicyArn,
			})
			if err != nil {
				fmt.Printf("   Warning: failed to detach policy %s: %v\n", *policy.PolicyArn, err)
			}

			// Delete custom policies (not AWS managed)
			if !isAWSManagedPolicy(*policy.PolicyArn) {
				_, err := d.iamClient.DeletePolicy(ctx, &iam.DeletePolicyInput{
					PolicyArn: policy.PolicyArn,
				})
				if err != nil {
					fmt.Printf("   Warning: failed to delete policy %s: %v\n", *policy.PolicyArn, err)
				} else {
					fmt.Printf("   Deleted policy: %s\n", *policy.PolicyArn)
				}
			}
		}
	}

	// Delete the role
	_, err = d.iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(d.roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete IAM role: %w", err)
	}

	fmt.Printf("   Deleted IAM role: %s\n", d.roleName)
	return nil
}

func isAWSManagedPolicy(arn string) bool {
	// AWS managed policies have a specific pattern
	return len(arn) > 20 && arn[13:26] == "arn:aws:iam::aws:policy/"
} 