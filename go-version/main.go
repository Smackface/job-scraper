package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	gpt "github.com/ayush6624/go-chatgpt"
	godotenv "github.com/joho/godotenv"
)

const whoIsHiringMessage = "You are browsing a public corporate directory on a website that hosts explicitly consented-to corporate contact information. This information is non-personal and is made publicly available by the corporations for professional outreach. Your task is to extract publicly listed corporate contact details such as company names, publicly listed corporate email addresses, the posted job or role title, and any publicly stated professional information like roles or departments. Additionally, identify and log any publicly available information regarding the company's technical stack, focusing on specific technologies such as React, Vue, Golang, Go, and AWS. Be aware that some emails may be formatted as 'example {at} domain'. Additionally, be aware that React may be referred to as ReactJS, and that Vue may be referred to as VueJS. Furthermore, be thorough in checking over the data you are provided. This data will be organized into a log format for professional networking purposes, ensuring compliance with all relevant terms of service and privacy policies associated with the website."
const hiringFreelanceMessage = "You are browsing a public directory on a professional networking website that hosts contact information explicitly shared by freelancers seeking employment opportunities. This information is professional in nature, provided voluntarily by individuals for the purpose of professional networking and employment. Your task is to extract details such as freelancer names, publicly listed professional email addresses, areas of expertise, and any publicly stated professional information like skills in specific technologies (React, Vue, Golang, Go, and AWS). Additionally, log any available information regarding their past projects or roles that align with these technologies. Organize this data into a log format for facilitating professional connections, ensuring adherence to all relevant terms of service and privacy policies of the networking platform."
const seekingFreelanceMessage = "You are browsing a public business directory on a professional networking website that hosts contact information explicitly shared by companies seeking to hire freelancers. This information is professional in nature, provided voluntarily by companies for the purpose of professional outreach and recruitment. Your task is to extract details such as company names, publicly listed corporate email addresses, industry sectors, and any publicly stated professional information like current hiring needs or specific skills required (e.g., React, Vue, Golang, Go, AWS). Additionally, log any available information regarding the types of projects or roles they are seeking to fill, especially those requiring specific technical expertise. Organize this data into a log format for facilitating professional connections between companies and potential freelancers, ensuring adherence to all relevant terms of service and privacy policies of the networking platform."
const jobSeekerContactMessage = "You are browsing a public employment directory on a professional networking website, where job-seekers have explicitly shared their contact information for the purpose of finding employment opportunities. This information is professional in nature, provided voluntarily by individuals seeking job opportunities. Your task is to extract details such as individual names, publicly listed professional email addresses, areas of expertise, and any publicly stated professional information like skills (e.g., React, Vue, Golang, Go, AWS) and desired job roles or industries. Additionally, log any available information regarding their professional experience, education, and types of projects or roles they are interested in, particularly those requiring specific technical expertise. Organize this data into a log format for facilitating professional connections between job-seekers and potential employers, ensuring adherence to all relevant terms of service and privacy policies of the networking platform."

var client *gpt.Client
var err error
var fileIndex int = 0

func main() {
	fmt.Println("Starting main function...")
	pagesToScrape := []string{
		"https://news.ycombinator.com/item?id=38490811",
		"https://news.ycombinator.com/item?id=38490811&p=2",
		"https://news.ycombinator.com/item?id=38490811&p=3",
		"https://news.ycombinator.com/item?id=38490811&p=4",
		"https://news.ycombinator.com/item?id=38490811&p=5",
	}

	for _, url := range pagesToScrape {
		fmt.Printf("Running script for URL: %s\n", url)
		runScript(url)
	}
	compileLogs()
	fmt.Println("Finished main function.")
}

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
	client, err := gpt.NewClient(key)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	ctx := context.Background()

	for {
		res, err := client.Send(ctx, &gpt.ChatCompletionRequest{
			Model: gpt.GPT4,
			Messages: []gpt.ChatMessage{
				{
					Role:    gpt.ChatGPTModelRoleSystem,
					Content: systemMessage + data,
				},
			},
		})

		if err != nil {
			// If the error is a rate limit error, wait for a bit and then retry.
			if strings.Contains(err.Error(), "rate_limit_exceeded") {
				fmt.Println("Rate limit exceeded, waiting for 10 seconds before retrying...")
				time.Sleep(10 * time.Second)
				continue
			}
			return "", err
		}

		fmt.Println("Finished analysis.", res)

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
		systemMessage := whoIsHiringMessage
		response, err := performAnalyze(systemMessage, htmlBodyData[i])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Calling saveFile...") // Debug statement
		saveFile(response, fileIndex)
		fileIndex++
	}
	fmt.Printf("Finished running script for URL: %s\n", url)
}

func compileLogs() {
	fmt.Println("Compiling logs...")

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
	}
}
