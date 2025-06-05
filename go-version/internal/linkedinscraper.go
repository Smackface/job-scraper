package internal_linkedin_scraper

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
	gpt "github.com/sashabaranov/go-openai"
)

var client *gpt.Client
var err error
var pageHTML string

// SignInLinkedIn performs LinkedIn login automation with debug logs
func SignInLinkedIn() {
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

	log.Println("[DEBUG] Creating new ChromeDP context...")
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var screenshotBuf []byte

	log.Println("[DEBUG] Starting LinkedIn login automation sequence...")
	err = chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Navigating to LinkedIn login page...")
			return nil
		}),
		chromedp.Navigate("https://www.linkedin.com/login"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Sleeping for 3 seconds to allow page to load...")
			return nil
		}),
		chromedp.Sleep(3*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Waiting for username input to be visible...")
			return nil
		}),
		chromedp.WaitVisible(`//input[@name='session_key' or @id='username']`, chromedp.BySearch),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Taking full screenshot of login page...")
			return nil
		}),
		chromedp.FullScreenshot(&screenshotBuf, 90),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if len(screenshotBuf) == 0 {
				log.Println("[ERROR] Screenshot buffer is empty after login page load")
				return fmt.Errorf("screenshot buffer is empty")
			}
			debugScreenshotFile := fmt.Sprintf("LinkedIn_Login_Page_%d.png", time.Now().Unix())
			log.Printf("[DEBUG] Saving login page screenshot to %s\n", debugScreenshotFile)
			if writeErr := os.WriteFile(debugScreenshotFile, screenshotBuf, 0644); writeErr != nil {
				log.Printf("[ERROR] Failed to save screenshot: %v", writeErr)
			}
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Sending keys to username input...")
			return nil
		}),
		chromedp.SendKeys(`//input[@name='session_key' or @id='username']`, os.Getenv("LINKEDIN_EMAIL"), chromedp.BySearch),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Sending keys to password input...")
			return nil
		}),
		chromedp.SendKeys(`//input[@name='session_password' or @id='password']`, os.Getenv("LINKEDIN_PASSWORD"), chromedp.BySearch),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Clicking submit/login button...")
			return nil
		}),
		chromedp.Click(`//button[@type='submit' or @data-litms-control-urn]`, chromedp.BySearch),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Sleeping for 5 seconds after login submit...")
			return nil
		}),
		chromedp.Sleep(5*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Taking full screenshot after login...")
			return nil
		}),
		chromedp.FullScreenshot(&screenshotBuf, 90),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if len(screenshotBuf) == 0 {
				log.Println("[ERROR] Screenshot buffer is empty after login")
				return fmt.Errorf("screenshot buffer is empty")
			}
			loginScreenshotFile := fmt.Sprintf("LinkedIn_Login_Page_%d.png", time.Now().Unix())
			log.Printf("[DEBUG] Saving login screenshot to %s\n", loginScreenshotFile)
			if writeErr := os.WriteFile(loginScreenshotFile, screenshotBuf, 0644); writeErr != nil {
				log.Printf("[ERROR] Failed to save screenshot: %v", writeErr)
			}
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Clicking Jobs link...")
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Grabbing all HTML from the page...")
			return nil
		}),
		chromedp.OuterHTML("html", &pageHTML, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if len(pageHTML) == 0 {
				log.Println("[ERROR] Page HTML is empty")
				return fmt.Errorf("page HTML is empty")
			}
			htmlFile := fmt.Sprintf("linkedin_page_%d.html", time.Now().Unix())
			log.Printf("[DEBUG] Saving page HTML to %s\n", htmlFile)
			if writeErr := os.WriteFile(htmlFile, []byte(pageHTML), 0644); writeErr != nil {
				log.Printf("[ERROR] Failed to save HTML file: %v", writeErr)
			}
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG 2] Clicking Jobs link...")
			return nil
		}),
		chromedp.Click(`//a[contains(@href, '/jobs/') and contains(text(), 'Jobs')]`, chromedp.BySearch),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Sleeping for 5 seconds after navigating to Jobs...")
			return nil
		}),
		chromedp.Sleep(5*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Sending keys to job search input (React)...")
			return nil
		}),
		chromedp.SendKeys(`//input[@aria-label="Search by title, skill, or company"]`, "React", chromedp.BySearch),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Clicking Search button...")
			return nil
		}),
		chromedp.Click(`//button[normalize-space(text())="Search"]`, chromedp.BySearch),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Sleeping for 5 seconds after job search...")
			return nil
		}),
		chromedp.Sleep(5*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[DEBUG] Taking full screenshot of job search results...")
			return nil
		}),
		chromedp.FullScreenshot(&screenshotBuf, 90),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if len(screenshotBuf) == 0 {
				log.Println("[ERROR] Screenshot buffer is empty after job search")
				return fmt.Errorf("screenshot buffer is empty")
			}
			jobSearchScreenshotFile := fmt.Sprintf("Job_Search_Page_%d.png", time.Now().Unix())
			log.Printf("[DEBUG] Saving job search screenshot to %s\n", jobSearchScreenshotFile)
			if writeErr := os.WriteFile(jobSearchScreenshotFile, screenshotBuf, 0644); writeErr != nil {
				log.Printf("[ERROR] Failed to save screenshot: %v", writeErr)
			}
			return nil
		}),
	)
	if err == nil {
		screenshotFile := fmt.Sprintf("linkedin_login_screenshot_%d.png", time.Now().Unix())
		log.Printf("[DEBUG] Saving final screenshot to %s\n", screenshotFile)
		if writeErr := os.WriteFile(screenshotFile, screenshotBuf, 0644); writeErr != nil {
			log.Printf("[ERROR] Failed to save screenshot: %v", writeErr)
		}
	}
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}
}
