package internal_linkedin_scraper

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	gpt "github.com/sashabaranov/go-openai"
    "github.com/tdewolff/minify/v2"
    "github.com/tdewolff/minify/v2/html"

)

const whoIsHiringMessage = "You are browsing a public corporate directory on a website that hosts explicitly consented-to corporate contact information. This information is non-personal and is made publicly available by the corporations for professional outreach. Your task is to extract publicly listed corporate contact details such as company names, publicly listed corporate email addresses, the posted job or role title, and any publicly stated professional information like roles or departments. Additionally, identify and log any publicly available information regarding the company's technical stack, focusing on specific technologies such as React, Vue, Golang, Go, and AWS. Be aware that some emails may be formatted as 'example {at} domain'. Additionally, be aware that React may be referred to as ReactJS, and that Vue may be referred to as VueJS. Furthermore, be thorough in checking over the data you are provided. Never make up any information. Do not include posts or roles that do not include the technologies used anywhere in the post. Do not include posts which themselves include C or any derivative of the C language such as C# or C++. Any framework is okay, TypeScript and JavaScript as generic technologies are okay. Do not include languages other than JavaScript, TypeScript, or Go. Ensure that the data you return is accurate. Be extra thorough in checking over contact information, and technological stack. This data will be organized into a log format for professional networking purposes, ensuring compliance with all relevant terms of service and privacy policies associated with the website."

var fileIndex int = 0

// Concurrent processing configuration
const (
	maxConcurrentRequests = 4      // Reduced from 12 to 4 to respect rate limits
	apiRequestDelay      = 2000    // Increased to 2000ms (2 seconds) between chunks
	preRequestDelay      = 500     // Keep at 500ms
	retryBaseDelay       = 1000    // Base delay for exponential backoff (1 second)
	maxRetryDelay        = 60000   // Max delay for exponential backoff (60 seconds)
)

// ChunkResult holds the result of processing a single chunk
type ChunkResult struct {
	Index   int
	Content string
	Error   error
}

// Keyword filtering configuration
var (
	// Technologies we want to include
	includeKeywords = []string{
		"react", "reactjs", "react.js",
		"vue", "vuejs", "vue.js",
		"go", "golang", "go-lang",
		"javascript", "js",
		"typescript", "ts",
		"aws", "amazon web services",
		"node", "nodejs", "node.js",
		"next", "nextjs", "next.js",
		"angular", "angularjs",
		"svelte", "sveltekit",
		"express", "expressjs",
		"postgresql", "postgres",
		"mongodb", "mongo",
		"docker",
		"kubernetes", "k8s",
		"graphql",
	}
	
	// Technologies/keywords we want to exclude - DISABLED for now
	/*
	excludeKeywords = []string{
		// C languages (using very specific patterns to avoid false positives)
		"\\bc language\\b", "\\bc programming\\b", "\\bc/c\\+\\+\\b", "\\bc \\+\\+\\b", "\\bc\\+\\+\\b", 
		"\\bc#\\b", "\\bc-sharp\\b", "csharp", "\\.net framework\\b", "\\.net core\\b", "\\.net\\b",
		"\\bjava programming\\b", "\\bjava development\\b", "\\bjava\\b", "\\bpython\\b", "django", "flask",
		"\\bruby\\b", "rails", "ruby on rails",
		"\\bphp\\b", "laravel", "symfony",
		"\\bswift\\b", "objective-c", "objective c",
		"\\bkotlin\\b",
		"\\bscala\\b",
		"\\bclojure\\b",
		"erlang", "elixir",
		"\\bhaskell\\b",
		"\\brust\\b",
		"\\bperl\\b",
		"r programming", "\\br language\\b", "\\br statistical\\b",
		"\\bmatlab\\b",
		"\\bcobol\\b",
		"\\bfortran\\b",
		"\\bassembly\\b",
	}
	*/
)

// getLogsDir returns the appropriate directory for storing logs
// Uses /tmp in AWS Lambda (read-write), logs/ locally
func getLogsDir() string {
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		return "/tmp"
	}
	return "logs"
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

	// fmt.Println(data, "Data")
	
	// Only load .env file when not running in AWS Lambda
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") == "" {
		log.Println("[DEBUG] Loading .env file (local environment)...")
		err := godotenv.Load()
		if err != nil {
			return "", fmt.Errorf("error loading .env file: %w", err)
		}
	} else {
		log.Println("[DEBUG] Skipping .env file (AWS Lambda environment)")
	}

	key := os.Getenv("OPENAI_KEY")
	if key == "" {
		return "", fmt.Errorf("OPENAI_KEY environment variable is not set or empty")
	}
	fmt.Println("API Key loaded (first 10 chars):", key[:min(10, len(key))])

	client := gpt.NewClient(key)
	ctx := context.Background()

	// Reduced delay before making request
	fmt.Printf("Waiting %.1f seconds before making API request...\n", float64(preRequestDelay)/1000)
	time.Sleep(time.Duration(preRequestDelay) * time.Millisecond)

	// Enhanced retry logic with exponential backoff
	maxRetries := 5 // Increased from 3 to 5
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
			
			// Check for rate limit errors
			if strings.Contains(err.Error(), "rate_limit_exceeded") ||
				strings.Contains(err.Error(), "429") ||
				strings.Contains(err.Error(), "Too Many Requests") ||
				strings.Contains(strings.ToLower(err.Error()), "rate limit") {

				if retries < maxRetries {
					retries++
					
					// Parse wait time from error message if available
					waitTime := parseRetryAfter(err.Error())
					if waitTime == 0 {
						// Exponential backoff: 1s, 2s, 4s, 8s, 16s
						waitTime = time.Duration(retryBaseDelay<<(retries-1)) * time.Millisecond
						if waitTime > time.Duration(maxRetryDelay)*time.Millisecond {
							waitTime = time.Duration(maxRetryDelay) * time.Millisecond
						}
					}
					
					fmt.Printf("🚨 Rate limit hit! Waiting %v before retry (attempt %d/%d)...\n", waitTime, retries, maxRetries)
					time.Sleep(waitTime)
					continue
				} else {
					return "", fmt.Errorf("max retries exceeded for rate limiting: %v", err)
				}
			}
			
			// For other errors, return immediately
			return "", err
		}

		fmt.Println("✅ API request successful!")

		// Check if there are any choices and if so, return the content of the first choice.
		if len(res.Choices) > 0 {
			return res.Choices[0].Message.Content, nil
		}

		// If there are no choices, return an empty string.
		return "", nil
	}
}

// parseRetryAfter extracts retry delay from OpenAI error messages
func parseRetryAfter(errorMsg string) time.Duration {
	// Look for patterns like "Please try again in 607ms" or "Please try again in 2.5s"
	if strings.Contains(errorMsg, "Please try again in") {
		// Extract the time value - this is a simplified parser
		if strings.Contains(errorMsg, "ms") {
			// Try to extract milliseconds
			parts := strings.Split(errorMsg, "Please try again in ")
			if len(parts) > 1 {
				timePart := strings.Split(parts[1], "ms")[0]
				timePart = strings.TrimSpace(timePart)
				if ms, err := time.ParseDuration(timePart + "ms"); err == nil {
					return ms
				}
			}
		} else if strings.Contains(errorMsg, "s") {
			// Try to extract seconds
			parts := strings.Split(errorMsg, "Please try again in ")
			if len(parts) > 1 {
				timePart := strings.Split(parts[1], "s")[0]
				timePart = strings.TrimSpace(timePart)
				if s, err := time.ParseDuration(timePart + "s"); err == nil {
					return s
				}
			}
		}
	}
	return 0 // Return 0 if we can't parse the retry time
}

// Note: Old saveFile function removed - now using thread-safe saveFileByIndex

func breakUpData(data string) ([]string, error) {
	fmt.Println("Breaking up data by job posting boundaries...")
	// Parse HTML and remove all <tr> elements whose DIRECT innerText includes "parent"
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	// Process from innermost to outermost to avoid parent-child conflicts, first pass
	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		// Get only the direct text content of this tr, excluding nested elements
		directText := ""
		s.Contents().Each(func(j int, content *goquery.Selection) {
			// Only collect text nodes that are direct children (not nested in other elements)
			if goquery.NodeName(content) == "#text" {
				directText += content.Text()
			}
		})
		
		// Also check immediate child elements (td, th) but not nested tr elements
		s.Children().Not("tr, table").Each(func(k int, child *goquery.Selection) {
			childText := child.Text()
			// Only add if this child doesn't contain nested tr elements
			if child.Find("tr").Length() == 0 {
				directText += " " + childText
			}
		})
		
		// Check if this tr should be removed based on its direct content only
		if strings.Contains(strings.ToLower(directText), "parent") {
			s.Remove()
		}
	})

	// Start to remove all undesired elements, second pass
	htmlTagsToRemove := []string {"span", "br", "img", "form", "u", "font"}
	for _, tag := range htmlTagsToRemove {
		doc.Find(tag).Each(func(i int, s *goquery.Selection) {
			s.Remove()
		})
	}
	
	htmlAfterCommentRemoval, err := doc.Html()
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML after comment removal: %w", err)
	}
	
	stringsToRemove := []string{
		"<[^>]*>?",
		"\n",
		"\\s+",
		"\\d+ points by \\w+ \\d+ days ago",
		"hide|past|favorite|\\d+&nbsp;comments",
		"class=\"comment\">",
		"<div>",
		"</div>",
		"<table>",
		"</table>",
		"<td>",
		"</td>",
		"<tbody>",
		"</tbody>",
		"<th>",
		"</th>",
		"<thead>",
		"</thead>",
		// HTML document structure tags
		"<html lang=\"en\" op=\"item\">",
		"</html>",
		"<head>",
		"</head>",
		"<body>",
		"</body>",
		"<title>",
		"</title>",
		// Meta and link tags
		"<meta name=\"referrer\" content=\"origin\"/>",
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"/>",
		"<link rel=\"stylesheet\" type=\"text/css\" href=\"news.css?c5rFJeaXqANRegCfZJSx\"/>",
		"<link rel=\"icon\" href=\"y18.svg\"/>",
		"<link rel=\"canonical\" href=\"https://news.ycombinator.com/item?id=44159528\"/>",
		// Layout tags
		"<center>",
		"</center>",
		"<p>",
		"</p>",
		"<i>",
		"</i>",
		"<pre>",
		"</pre>",
		"<code>",
		"</code>",
		// Table structure tags (excluding tr)
		"<table id=\"hnmain\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" width=\"85%\" bgcolor=\"#f6f6ef\">",
		"<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" width=\"100%\" style=\"padding:2px\">",
		"<table class=\"fatitem\" border=\"0\">",
		"<table border=\"0\" class=\"comment-tree\">",
		"<table border=\"0\">",
		// TD with various attributes
		"<td bgcolor=\"#ff6600\">",
		"<td style=\"width:18px;padding-right:4px\">",
		"<td style=\"line-height:12pt; height:10px;\">",
		"<td style=\"text-align:right;padding-right:4px;\">",
		"<td align=\"right\" valign=\"top\" class=\"title\">",
		"<td valign=\"top\" class=\"votelinks\">",
		"<td class=\"title\">",
		"<td colspan=\"2\">",
		"<td class=\"subtext\">",
		"<td style=\"height:2px\">",
		"<td style=\"height:10px\">",
		"<td class=\"ind\" indent=\"0\">",
		"<td valign=\"top\" class=\"votelinks\">",
		"<td class=\"default\">",
		// Div with various classes and styles
		"<div class=\"toptext\">",
		"<div style=\"margin-top:2px; margin-bottom:-10px;\">",
		"<div class=\"commtext c00\">",
		"<div class=\"reply\">",
		// Additional common tags
		"border=\"0\"",
		"cellpadding=\"0\"",
		"cellspacing=\"0\"",
		"width=\"85%\"",
		"width=\"100%\"",
		"bgcolor=\"#f6f6ef\"",
		"bgcolor=\"#ff6600\"",
		"class=\"athing submission\"",
		"class=\"athing comtr\"",
		"class=\"fatitem\"",
		"class=\"comment-tree\"",
		"class=\"title\"",
		"class=\"subtext\"",
		"class=\"votelinks\"",
		"class=\"default\"",
		"class=\"toptext\"",
		"class=\"commtext c00\"",
		"class=\"reply\"",
		"class=\"ind\"",
		"id=\"44159528\"",
		"id=\"pagespace\"",
		"title=\"Ask HN: Who is hiring? (June 2025)\"",
		"style=\"height:10px\"",
		"style=\"padding:2px\"",
		"style=\"width:18px;padding-right:4px\"",
		"style=\"line-height:12pt; height:10px;\"",
		"style=\"text-align:right;padding-right:4px;\"",
		"style=\"height:2px\"",
		"style=\"margin-top:2px; margin-bottom:-10px;\"",
		"align=\"right\"",
		"valign=\"top\"",
		"indent=\"0\"",
		"colspan=\"2\"",
		"<div",
		"rel=\"nofollow\"",
		"rel=\"noopener noreferrer\"",
		"rel=\"noopener\"",
		"rel=\"noreferrer\"",
		"rel=\"noreferrer nofollow\"",
	}
	
	reformattedData := htmlAfterCommentRemoval
	for _, str := range stringsToRemove {
		reformattedData = strings.ReplaceAll(reformattedData, str, "")
	}
	
	newDoc, err := goquery.NewDocumentFromReader(strings.NewReader(reformattedData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse reformatted data: %w", err)
	}
	
	// Remove all tr elements who have no children
	newDoc.Find("tr").Each(func(i int, s *goquery.Selection) {
		if s.Children().Length() == 0 {
			s.Remove()
		}
	})

	reformattedData, err = newDoc.Html()
	if err != nil {
		return nil, fmt.Errorf("failed to get reformatted data: %w", err)
	}

	reformattedData, err = customMinifyHTML().String("text/html", reformattedData)
	if err != nil {
		return nil, fmt.Errorf("failed to minify HTML: %w", err)
	}

	// New intelligent chunking by job posting boundaries
	chunks, err := chunkByJobPostings(reformattedData)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk by job postings: %w", err)
	}

	// Write chunks to files for debugging
	// for i, chunk := range chunks {
	// 	if err := os.WriteFile(fmt.Sprintf("dataChunk-%d.html", i), []byte(chunk), 0644); err != nil {
	// 		return nil, fmt.Errorf("failed to write dataChunk-%d.html: %w", i, err)
	// 	}
	// }

	fmt.Printf("Finished breaking up data into %d intelligent chunks.\n", len(chunks))
	return chunks, nil
}

// chunkByJobPostings splits data by job posting boundaries to preserve context
func chunkByJobPostings(data string) ([]string, error) {
	// Split by job posting boundaries using the vote link pattern
	// Each job posting starts with <a id=up_XXXXXXX
	jobPostingPattern := `<a id=up_\d+`
	re := regexp.MustCompile(jobPostingPattern)
	
	// Find all job posting start positions
	matches := re.FindAllStringIndex(data, -1)
	
	if len(matches) == 0 {
		// Fallback to simple chunking if no pattern found
		fmt.Println("No job posting patterns found, using simple chunking")
		return simpleChunk(data, 30000), nil // Updated to 30KB to stay under token limit
	}
	
	var chunks []string
	maxChunkSize := 30000 // Reduced to 30KB to stay under GPT-4's 8,192 token limit
	
	fmt.Printf("Found %d job postings to chunk\n", len(matches))
	
	for i := 0; i < len(matches); i++ {
		start := matches[i][0]
		var end int
		if i < len(matches)-1 {
			end = matches[i+1][0]
		} else {
			end = len(data)
		}
		
		jobPosting := data[start:end]
		
		// If a single job posting is too large, split it intelligently
		if len(jobPosting) > maxChunkSize {
			subChunks := splitLargeJobPosting(jobPosting, maxChunkSize)
			chunks = append(chunks, subChunks...)
		} else {
			chunks = append(chunks, jobPosting)
		}
	}
	
	// Combine small adjacent chunks to optimize API usage and get closer to 30KB
	chunks = combineSmallChunks(chunks, maxChunkSize)
	
	return chunks, nil
}

// splitLargeJobPosting splits very large job postings while preserving context
func splitLargeJobPosting(jobPosting string, maxSize int) []string {
	// For very large job postings, split by sentences or paragraphs
	// while keeping context together
	sentences := strings.Split(jobPosting, ". ")
	var chunks []string
	var currentChunk strings.Builder
	
	for _, sentence := range sentences {
		// Add sentence if it fits, otherwise start new chunk
		if currentChunk.Len()+len(sentence)+2 > maxSize && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}
		currentChunk.WriteString(sentence)
		if !strings.HasSuffix(sentence, ".") {
			currentChunk.WriteString(". ")
		}
	}
	
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	
	return chunks
}

// combineSmallChunks combines small job postings to optimize chunk size (targeting 30KB)
func combineSmallChunks(chunks []string, maxSize int) []string {
	var combined []string
	var currentCombined strings.Builder
	
	fmt.Printf("🔧 Combining chunks to target %d bytes (%.1fKB)...\n", maxSize, float64(maxSize)/1024)
	
	for i, chunk := range chunks {
		// Try to add chunk to current combined chunk
		newSize := currentCombined.Len() + len(chunk) + 4 // +4 for separator
		
		if newSize <= maxSize {
			if currentCombined.Len() > 0 {
				currentCombined.WriteString("\n\n") // Separator between job postings
			}
			currentCombined.WriteString(chunk)
			fmt.Printf("  📝 Added job posting %d to current chunk (now %d bytes)\n", i+1, currentCombined.Len())
		} else {
			// Current chunk would exceed limit, finalize it
			if currentCombined.Len() > 0 {
				finalSize := currentCombined.Len()
				combined = append(combined, currentCombined.String())
				fmt.Printf("  ✅ Finalized chunk #%d: %d bytes (%.1fKB)\n", len(combined), finalSize, float64(finalSize)/1024)
				currentCombined.Reset()
			}
			// Start new chunk with current job posting
			currentCombined.WriteString(chunk)
			fmt.Printf("  🆕 Started new chunk with job posting %d (%d bytes)\n", i+1, len(chunk))
		}
	}
	
	// Don't forget the last chunk
	if currentCombined.Len() > 0 {
		finalSize := currentCombined.Len()
		combined = append(combined, currentCombined.String())
		fmt.Printf("  ✅ Finalized final chunk #%d: %d bytes (%.1fKB)\n", len(combined), finalSize, float64(finalSize)/1024)
	}
	
	// Report final statistics
	totalOriginal := len(chunks)
	totalCombined := len(combined)
	fmt.Printf("📊 Chunk combination complete: %d job postings → %d chunks (%.1f%% reduction)\n", 
		totalOriginal, totalCombined, float64(totalOriginal-totalCombined)/float64(totalOriginal)*100)
		
	// Show size distribution
	for i, chunk := range combined {
		size := len(chunk)
		fmt.Printf("  📦 Chunk %d: %d bytes (%.1fKB) - %.1f%% of target\n", 
			i+1, size, float64(size)/1024, float64(size)/float64(maxSize)*100)
	}
	
	return combined
}

// simpleChunk provides fallback chunking if pattern matching fails (updated to 30KB)
func simpleChunk(data string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// func minifyHTML(htmlContent string) (string, error) {
// 	m := minify.New()
// 	m.AddFunc("text/html", html.Minify)

// 	minified, err := m.String("text/html", htmlContent)
// 	if err != nil {
// 		return "", err
// 	}
// 	return minified, nil
// }

func customMinifyHTML() *minify.M {
	m := minify.New()

	// custom minification, let's really squeeze as much juice out as we possibly can
	m.Add("text/html", &html.Minifier{
		KeepComments: false,
		KeepConditionalComments: false,
		KeepSpecialComments: false,
		KeepDefaultAttrVals: false,
		KeepDocumentTags: false,
		KeepEndTags: false,
		KeepQuotes: false,
		KeepWhitespace: false,
		TemplateDelims: [2]string{"", ""},
	})
	return m
}

// filterChunksByKeywords filters chunks to only include those with relevant technologies
func filterChunksByKeywords(chunks []string) []string {
	var filteredChunks []string
	var rejectedCount int
	
	fmt.Printf("🔍 Filtering %d chunks by technology keywords...\n", len(chunks))
	
	for i, chunk := range chunks {
		chunkLower := strings.ToLower(chunk)
		
		// Check for include keywords only (removed exclusion filtering)
		hasRelevantTech := false
		var foundKeywords []string
		for _, includeWord := range includeKeywords {
			if strings.Contains(chunkLower, strings.ToLower(includeWord)) {
				hasRelevantTech = true
				foundKeywords = append(foundKeywords, includeWord)
			}
		}
		
		if hasRelevantTech {
			filteredChunks = append(filteredChunks, chunk)
			fmt.Printf("✅ Chunk %d accepted: found keywords %v\n", i+1, foundKeywords)
		} else {
			rejectedCount++
			fmt.Printf("⚪ Chunk %d rejected: no relevant technology keywords found\n", i+1)
		}
	}
	
	fmt.Printf("📊 Filtering complete: %d chunks accepted, %d rejected (%.1f%% reduction)\n", 
		len(filteredChunks), rejectedCount, float64(rejectedCount)/float64(len(chunks))*100)
	
	return filteredChunks
}

// processChunksConcurrently processes chunks in parallel with controlled concurrency
func processChunksConcurrently(chunks []string, systemMessage string) ([]string, error) {
	numChunks := len(chunks)
	fmt.Printf("🚀 Starting concurrent processing of %d chunks with max %d concurrent requests\n", numChunks, maxConcurrentRequests)
	
	// Create semaphore to limit concurrent requests
	semaphore := make(chan struct{}, maxConcurrentRequests)
	
	// Channel to collect results
	resultChan := make(chan ChunkResult, numChunks)
	
	var wg sync.WaitGroup
	
	// Process each chunk concurrently
	for i, chunk := range chunks {
		wg.Add(1)
		go func(index int, data string) {
			defer wg.Done()
			
			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Add staggered delay to spread out requests and avoid rate limits
			// Increased stagger delay to reduce rate limit pressure
			staggerDelay := time.Duration(index*500) * time.Millisecond // Increased from 100ms to 500ms
			if staggerDelay > 0 {
				fmt.Printf("⏳ Chunk %d: Staggering request by %v to avoid rate limits\n", index+1, staggerDelay)
				time.Sleep(staggerDelay)
			}
			
			fmt.Printf("🔄 Processing chunk %d/%d...\n", index+1, numChunks)
			
			result, err := performAnalyze(systemMessage, data)
			resultChan <- ChunkResult{
				Index:   index,
				Content: result,
				Error:   err,
			}
			
			if err != nil {
				fmt.Printf("❌ Chunk %d failed: %v\n", index+1, err)
			} else {
				fmt.Printf("✅ Chunk %d completed successfully\n", index+1)
			}
		}(i, chunk)
	}
	
	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results maintaining order
	results := make([]string, numChunks)
	var errors []string
	
	for result := range resultChan {
		if result.Error != nil {
			errors = append(errors, fmt.Sprintf("chunk %d: %v", result.Index+1, result.Error))
		} else {
			results[result.Index] = result.Content
		}
	}
	
	// Check if we had any errors
	if len(errors) > 0 {
		return nil, fmt.Errorf("errors processing chunks: %s", strings.Join(errors, "; "))
	}
	
	fmt.Printf("🎉 All %d chunks processed successfully!\n", numChunks)
	return results, nil
}

// saveResultsToFiles saves the processed results to individual files
func saveResultsToFiles(results []string) error {
	fmt.Printf("💾 Saving %d results to files...\n", len(results))
	
	for i, content := range results {
		if content == "" {
			fmt.Printf("⚠️  Skipping empty result for chunk %d\n", i+1)
			continue
		}
		
		err := saveFileByIndex(content, fileIndex)
		if err != nil {
			return fmt.Errorf("failed to save file %d: %w", fileIndex, err)
		}
		fileIndex++
	}
	
	fmt.Printf("✅ Successfully saved %d files\n", fileIndex)
	return nil
}

// saveFileByIndex saves content to a file with the given index (thread-safe version)
func saveFileByIndex(content string, index int) error {
	fmt.Printf("Saving file number %d\n", index)

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(getLogsDir(), 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	file, err := os.Create(fmt.Sprintf("%s/output-%d.log", getLogsDir(), index))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	
	fmt.Printf("Finished saving file number %d\n", index)
	return nil
}

func runScript(url string) error {
	fmt.Printf("Running script for URL: %s\n", url)
	data, err := fetchHackerNews(url)
	if err != nil {
		return fmt.Errorf("failed to fetch data: %w", err)
	}
	
	htmlBodyData, err := breakUpData(data)
	if err != nil {
		return fmt.Errorf("failed to break up data: %w", err)
	}
	
	fmt.Printf("📊 Data broken into %d chunks\n", len(htmlBodyData))
	
	// Filter chunks by keywords
	filteredChunks := filterChunksByKeywords(htmlBodyData)
	
	// Process chunks concurrently
	systemMessage := whoIsHiringMessage
	results, err := processChunksConcurrently(filteredChunks, systemMessage)
	if err != nil {
		return fmt.Errorf("failed to process chunks: %w", err)
	}
	
	// Save results to files
	err = saveResultsToFiles(results)
	if err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}
	
	fmt.Printf("Finished running script for URL: %s\n", url)
	return nil
}

func compileLogs() {
	fmt.Println("Compiling logs...")

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(getLogsDir(), 0755); err != nil {
		log.Fatal(err)
	}

	outputFile, err := os.Create(fmt.Sprintf("%s/compiled.log", getLogsDir()))
	if err != nil {
		log.Fatal(err)
	}
	defer outputFile.Close()

	for i := 0; i < fileIndex; i++ {
		logFile, err := os.Open(fmt.Sprintf("%s/output-%d.log", getLogsDir(), i))
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

// consolidate logs into a single file and then return the .html file for download by end-user in the browser
// we handle the return of the file in the lambda function
// when running this locally, we just consolidate the logs (without deleting individual files)
func consolidateLogs() (string, error) {
	fmt.Println("Consolidating logs...")

	logsDir := getLogsDir()
	compiledFile := fmt.Sprintf("%s/compiled.log", logsDir)

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
		if file.Name() == "compiled.log" {
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
		if file.Name() == "compiled.log" {
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
	// Only load .env file when not running in AWS Lambda
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") == "" {
		log.Println("[DEBUG] Loading .env file (local environment)...")
		err := godotenv.Load()
		if err != nil {
			log.Fatal("[ERROR] Error loading .env file")
		}
	} else {
		log.Println("[DEBUG] Skipping .env file (AWS Lambda environment)")
	}

	for _, url := range pagesToScrape {
		fmt.Printf("Running script for URL: %s\n", url)
		err := runScript(url)
		if err != nil {
			log.Fatal(err)
		}
	}
	compileLogs()
	fileToReturn, err := consolidateLogs()
	if err != nil {
		log.Fatal(err)
	}
	return fileToReturn, nil
}
