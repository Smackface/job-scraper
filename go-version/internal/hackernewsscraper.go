package internal_linkedin_scraper

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	gpt "github.com/sashabaranov/go-openai"

)

const whoIsHiringMessage = "You are browsing a public corporate directory on a website that hosts explicitly consented-to corporate contact information. This information is non-personal and is made publicly available by the corporations for professional outreach. Your task is to extract publicly listed corporate contact details such as company names, publicly listed corporate email addresses, the posted job or role title, and any publicly stated professional information like roles or departments. Additionally, identify and log any publicly available information regarding the company's technical stack, focusing on specific technologies such as React, Vue, Golang, Go, and AWS. Be aware that some emails may be formatted as 'example {at} domain'. Additionally, be aware that React may be referred to as ReactJS, and that Vue may be referred to as VueJS. Furthermore, be thorough in checking over the data you are provided. Never make up any information. Do not include posts or roles that do not include the technologies used anywhere in the post. Do not include posts which themselves include C or any derivative of the C language such as C# or C++. Any framework is okay, TypeScript and JavaScript as generic technologies are okay. Do not include languages other than JavaScript, TypeScript, or Go. Ensure that the data you return is accurate. Be extra thorough in checking over contact information, and technological stack. This data will be organized into a log format for professional networking purposes, ensuring compliance with all relevant terms of service and privacy policies associated with the website."

var fileIndex int = 0

func fetchHackerNews(url string) (string, error) {
	fmt.Printf("Fetching data from URL: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Finished fetching data.")
	return string(body), nil
}

func performAnalyze(systemMessage string, data string) (string, error) {
	fmt.Println("Starting analysis...")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	key := os.Getenv("OPENAI_KEY")
	if key == "" {
		log.Fatal("OPENAI_KEY environment variable is not set or empty")
	}
	fmt.Println("API Key loaded (first 10 chars):", key[:min(10, len(key))])

	client := gpt.NewClient(key)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	ctx := context.Background()

	// Add delay before making request to avoid hitting rate limits
	fmt.Println("Waiting 2 seconds before making API request...")
	time.Sleep(2 * time.Second)

	// Move retry logic outside the error handling to prevent reset
	maxRetries := 3
	retries := 0

	for {
		fmt.Printf("Making API request (attempt %d/%d)...\n", retries+1, maxRetries+1)
		res, err := client.CreateChatCompletion(ctx, gpt.ChatCompletionRequest{
			Model: gpt.GPT4,
			Messages: []gpt.ChatCompletionMessage{
				{
					Role:    gpt.ChatMessageRoleSystem,
					Content: systemMessage + data,
				},
			},
		})

		if err != nil {
			fmt.Printf("API Error: %v\n", err)
			// Check for various rate limit error conditions
			if strings.Contains(err.Error(), "rate_limit_exceeded") ||
				strings.Contains(err.Error(), "429") ||
				strings.Contains(err.Error(), "Too Many Requests") ||
				strings.Contains(strings.ToLower(err.Error()), "rate limit") {

				if retries < maxRetries {
					retries++
					fmt.Printf("Rate limit exceeded, waiting for 30 seconds before retrying (attempt %d/%d)...\n", retries, maxRetries)
					time.Sleep(30 * time.Second)
					continue
				} else {
					return "", fmt.Errorf("max retries exceeded for rate limiting: %v", err)
				}
			}
			return "", err
		}

		fmt.Println("API request successful!")

		// Check if there are any choices and if so, return the content of the first choice.
		if len(res.Choices) > 0 {
			return res.Choices[0].Message.Content, nil
		}

		// If there are no choices, return an empty string.
		return "", nil
	}
}

func saveFile(content string, i int) {
	fmt.Printf("Saving file number %d\n", i)

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatal(err)
	}

	file, err := os.Create(fmt.Sprintf("logs/output-%d.log", i))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	_, err = file.WriteString(content)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Finished saving file number %d\n", i)
}

func breakUpData(data string) ([]string, error) {
	fmt.Println("Breaking up data...")
	// fmt.Println("Data: ", data)
	const chunkSize = 15000
	reformattedData := strings.ReplaceAll(data, "<[^>]*>?", "")
	reformattedData = strings.ReplaceAll(reformattedData, "\n", "")
	reformattedData = strings.ReplaceAll(reformattedData, "\\s+", " ")
	reformattedData = strings.ReplaceAll(reformattedData, "\\d+ points by \\w+ \\d+ days ago", "")
	reformattedData = strings.ReplaceAll(reformattedData, "hide|past|favorite|\\d+&nbsp;comments", "")
	// fmt.Println("Reformatted data: ", reformattedData)
	dataChunk := []string{}
	for i := 0; i < len(reformattedData); i += chunkSize {
		fmt.Println(i, chunkSize, len(reformattedData))
		dataChunk = append(dataChunk, reformattedData[i:min(i+chunkSize, len(reformattedData))])
	}
	fmt.Println(dataChunk)
	fmt.Println("Finished breaking up data.")
	return dataChunk, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func runScript(url string) {
	fmt.Printf("Running script for URL: %s\n", url)
	data, err := fetchHackerNews(url)
	if err != nil {
		log.Fatal(err)
	}
	htmlBodyData, newErr := breakUpData(data)
	if newErr != nil {
		log.Fatal(newErr)
	}
	fmt.Printf("htmlBodyData length: %d\n", len(htmlBodyData)) // Debug statement
	for i := range htmlBodyData {
		fmt.Printf("Processing chunk %d of %d...\n", i+1, len(htmlBodyData))
		systemMessage := whoIsHiringMessage
		response, err := performAnalyze(systemMessage, htmlBodyData[i])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Calling saveFile...") // Debug statement
		saveFile(response, fileIndex)
		fileIndex++

		// Add delay between chunks to avoid overwhelming the API
		if i < len(htmlBodyData)-1 { // Don't wait after the last chunk
			fmt.Println("Waiting 5 seconds before processing next chunk...")
			time.Sleep(5 * time.Second)
		}
	}
	fmt.Printf("Finished running script for URL: %s\n", url)
}

func compileLogs() {
	fmt.Println("Compiling logs...")

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatal(err)
	}

	outputFile, err := os.Create("logs/compiled.log")
	if err != nil {
		log.Fatal(err)
	}
	defer outputFile.Close()

	for i := 0; i < fileIndex; i++ {
		logFile, err := os.Open(fmt.Sprintf("logs/output-%d.log", i))
		if err != nil {
			log.Fatal(err)
		}

		logData, err := io.ReadAll(logFile)
		if err != nil {
			log.Fatal(err)
		}
		logFile.Close()

		_, err = outputFile.Write(logData)
		if err != nil {
			log.Fatal(err)
		}

		_, err = outputFile.WriteString("\n")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Finished compiling logs.")
		consolidateLogs()
	}
}

// consolidate logs into a single file and then return the .txt file for download by end-user in the browser
// we handle the return of the file in the lambda function
// when running this locally, we just consolidate the logs and then delete all the prior logs
func consolidateLogs() (string, error) {
	fmt.Println("Consolidating logs...")

	const logsDir = "logs"
	const compiledFile = "logs/compiled.txt"

	outputFile, err := os.Create(compiledFile)
	if err != nil {
		return "", fmt.Errorf("failed to create compiled log file: %w", err)
	}
	defer outputFile.Close()

	files, err := os.ReadDir(logsDir)
	if err != nil {
		return "", fmt.Errorf("failed to read logs directory: %w", err)
	}

	for _, file := range files {
		// Skip the compiled file itself if it exists in the directory
		if file.Name() == "compiled.txt" {
			continue
		}
		filePath := fmt.Sprintf("%s/%s", logsDir, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read log file %s: %w", file.Name(), err)
		}
		if _, err = outputFile.Write(content); err != nil {
			return "", fmt.Errorf("failed to write to compiled log file: %w", err)
		}
		if _, err = outputFile.WriteString("\n"); err != nil {
			return "", fmt.Errorf("failed to write newline to compiled log file: %w", err)
		}
	}

	fmt.Println("Finished consolidating logs.")

	// Delete all prior logs except the compiled file
	for _, file := range files {
		if file.Name() == "compiled.txt" {
			continue
		}
		filePath := fmt.Sprintf("%s/%s", logsDir, file.Name())
		if err := os.Remove(filePath); err != nil {
			return "", fmt.Errorf("failed to delete log file %s: %w", file.Name(), err)
		}
	}

	return compiledFile, nil
}

func ScrapeHackerNews(pagesToScrape []string) (string, error) {
	log.Println("[DEBUG] Loading .env file...")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("[ERROR] Error loading .env file")
	}

	for _, url := range pagesToScrape {
		fmt.Printf("Running script for URL: %s\n", url)
		runScript(url)
	}
	compileLogs()
	fileToReturn, err := consolidateLogs()
	if err != nil {
		log.Fatal(err)
	}
	return fileToReturn, nil
}
