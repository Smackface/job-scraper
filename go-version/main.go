package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/joho/godotenv"
	gpt "github.com/sashabaranov/go-openai"

	internal_hackernewsscraper "github.com/Smackface/go-job-scraper/internal"
)

const whoIsHiringMessage = "You are browsing a public corporate directory on a website that hosts explicitly consented-to corporate contact information. This information is non-personal and is made publicly available by the corporations for professional outreach. Your task is to extract publicly listed corporate contact details such as company names, publicly listed corporate email addresses, the posted job or role title, and any publicly stated professional information like roles or departments. Additionally, identify and log any publicly available information regarding the company's technical stack, focusing on specific technologies such as React, Vue, Golang, Go, and AWS. Be aware that some emails may be formatted as 'example {at} domain'. Additionally, be aware that React may be referred to as ReactJS, and that Vue may be referred to as VueJS. Furthermore, be thorough in checking over the data you are provided. This data will be organized into a log format for professional networking purposes, ensuring compliance with all relevant terms of service and privacy policies associated with the website."
const hiringFreelanceMessage = "You are browsing a public directory on a professional networking website that hosts contact information explicitly shared by freelancers seeking employment opportunities. This information is professional in nature, provided voluntarily by individuals for the purpose of professional networking and employment. Your task is to extract details such as freelancer names, publicly listed professional email addresses, areas of expertise, and any publicly stated professional information like skills in specific technologies (React, Vue, Golang, Go, and AWS). Additionally, log any available information regarding their past projects or roles that align with these technologies. Organize this data into a log format for facilitating professional connections, ensuring adherence to all relevant terms of service and privacy policies of the networking platform."
const seekingFreelanceMessage = "You are browsing a public business directory on a professional networking website that hosts contact information explicitly shared by companies seeking to hire freelancers. This information is professional in nature, provided voluntarily by companies for the purpose of professional outreach and recruitment. Your task is to extract details such as company names, publicly listed corporate email addresses, industry sectors, and any publicly stated professional information like current hiring needs or specific skills required (e.g., React, Vue, Golang, Go, AWS). Additionally, log any available information regarding the types of projects or roles they are seeking to fill, especially those requiring specific technical expertise. Organize this data into a log format for facilitating professional connections between companies and potential freelancers, ensuring adherence to all relevant terms of service and privacy policies of the networking platform."
const jobSeekerContactMessage = "You are browsing a public employment directory on a professional networking website, where job-seekers have explicitly shared their contact information for the purpose of finding employment opportunities. This information is professional in nature, provided voluntarily by individuals seeking job opportunities. Your task is to extract details such as individual names, publicly listed professional email addresses, areas of expertise, and any publicly stated professional information like skills (e.g., React, Vue, Golang, Go, AWS) and desired job roles or industries. Additionally, log any available information regarding their professional experience, education, and types of projects or roles they are interested in, particularly those requiring specific technical expertise. Organize this data into a log format for facilitating professional connections between job-seekers and potential employers, ensuring adherence to all relevant terms of service and privacy policies of the networking platform."

var client *gpt.Client
var err error
var fileIndex int = 0

type TRequest = events.APIGatewayProxyRequest
type TResponse = events.APIGatewayProxyResponse

// Define a type for the POST body, which will be a JSON array of URLs to scrape.
type ScrapeRequestBody []string

// File download response structure for automatic downloads
type FileDownloadResponse struct {
	Message     string `json:"message"`
	Scraper     string `json:"scraper"`
	FileName    string `json:"fileName"`
	FileContent string `json:"fileContent"` // Base64 encoded
	ContentType string `json:"contentType"`
	FileSize    int64  `json:"fileSize"`
	Success     bool   `json:"success"`
}

// Define differentiated POST request templates for each scraper service.
// Each request includes a query string parameter "scraper" to specify the scraper type.

var ScrapeHackerNewsRequest = TRequest{
	Body: "",
	Headers: map[string]string{
		"Content-Type": "application/json",
	},
	IsBase64Encoded: false,
	Path:            "/scrape",
	HTTPMethod:      "POST",
	QueryStringParameters: map[string]string{
		"scraper": "hackernews",
	},
}

var ScrapeLinkedInRequest = TRequest{
	Body: "",
	Headers: map[string]string{
		"Content-Type": "application/json",
	},
	IsBase64Encoded: false,
	Path:            "/scrape",
	HTTPMethod:      "POST",
	QueryStringParameters: map[string]string{
		"scraper": "linkedin",
	},
}

// Lambda handler function
func lambdaHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Println("Lambda handler invoked...")

	// Instantiate pages to scrape using the request body
	var pagesToScrape []string
	if err := json.Unmarshal([]byte(request.Body), &pagesToScrape); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: `{"message": "Invalid request body. Expected a JSON array of strings.", "success": false}`,
		}, nil
	}

	// Determine scraper type from query parameters
	scraperType := request.QueryStringParameters["scraper"]

	switch scraperType {
	case "hackernews":
		fmt.Println("Running Hacker News scraper...")
		filePath, err := internal_hackernewsscraper.ScrapeHackerNews(pagesToScrape)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Headers: map[string]string{
					"Content-Type":                "application/json",
					"Access-Control-Allow-Origin": "*",
				},
				Body: fmt.Sprintf(`{"message": "Error scraping Hacker News: %v", "success": false}`, err),
			}, nil
		}

		// Read the file content
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Headers: map[string]string{
					"Content-Type":                "application/json",
					"Access-Control-Allow-Origin": "*",
				},
				Body: fmt.Sprintf(`{"message": "Error reading file: %v", "success": false}`, err),
			}, nil
		}

		// Get file info
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Headers: map[string]string{
					"Content-Type":                "application/json",
					"Access-Control-Allow-Origin": "*",
				},
				Body: fmt.Sprintf(`{"message": "Error getting file info: %v", "success": false}`, err),
			}, nil
		}

		// Create filename with timestamp
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		fileName := fmt.Sprintf("hackernews_jobs_%s.log", timestamp)

		// Base64 encode the file content
		encodedContent := base64.StdEncoding.EncodeToString(fileContent)

		// Create response with file data
		response := FileDownloadResponse{
			Message:     "Hacker News scraping completed successfully",
			Scraper:     "hackernews",
			FileName:    fileName,
			FileContent: encodedContent,
			ContentType: "text/plain",
			FileSize:    fileInfo.Size(),
			Success:     true,
		}

		responseBody, err := json.Marshal(response)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Headers: map[string]string{
					"Content-Type":                "application/json",
					"Access-Control-Allow-Origin": "*",
				},
				Body: fmt.Sprintf(`{"message": "Error creating response: %v", "success": false}`, err),
			}, nil
		}

		// Clean up the temporary file
		os.Remove(filePath)

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type":                 "application/json",
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Headers": "Content-Type",
				"Access-Control-Allow-Methods": "POST, GET, OPTIONS",
			},
			Body: string(responseBody),
		}, nil

	case "linkedin":
		return events.APIGatewayProxyResponse{
			StatusCode: 501,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: `{"message": "LinkedIn scraper functionality is currently unavailable pending clarification of LinkedIn's Terms of Service.", "success": false}`,
		}, nil

	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: `{"message": "Invalid or missing scraper parameter. Use 'hackernews' or 'linkedin'", "success": false}`,
		}, nil
	}
}

func main() {
	loadConfig()
	godotenv.Load()
	internal_hackernewsscraper.Run([]string{"https://news.ycombinator.com/item?id=44159528"})
	// fmt.Println("Starting main function...")

	// // Check if running in AWS Lambda environment
	// // AWS Lambda sets the AWS_LAMBDA_RUNTIME_API environment variable
	// isAWSLambda := os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""

	// if isAWSLambda {
	// 	fmt.Println("Running in AWS Lambda environment...")
	// 	// Start the Lambda handler
	// 	lambda.Start(lambdaHandler)
	// } else {
	// 	fmt.Println("Running locally...")
	// 	// Parse CLI flags for local execution
	// 	scrapeHackerNews := flag.Bool("scrape-hacker-news", false, "Scrape Hacker News")
	// 	flag.Parse()

	// 	// Instantiate pages to scrape
	// 	pagesToScrape := []string{
	// 		"https://news.ycombinator.com/item?id=44159528",
	// 		"https://news.ycombinator.com/item?id=44159528&p=2",
	// 		"https://news.ycombinator.com/item?id=44159528&p=3",
	// 		"https://news.ycombinator.com/item?id=44159528&p=4",
	// 		"https://news.ycombinator.com/item?id=44159528&p=5",
	// 	}

	// 	if *scrapeHackerNews {
	// 		fmt.Println("Running Hacker News scraper locally...")
	// 		internal_hackernewsscraper.ScrapeHackerNews(pagesToScrape)
	// 		fmt.Println("Hacker News scraping completed.")
	// 	} else {
	// 		fmt.Println("No scraper specified. Use --scrape-hacker-news flag")
	// 	}
	// }

	// fmt.Println("Finished main function.")
}
