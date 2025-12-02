//go:build ignore
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/diogo/geminiweb/internal/api"
	"github.com/diogo/geminiweb/internal/config"
	"github.com/diogo/geminiweb/internal/models"
)

func main() {
	fmt.Println("=== Final Debug Test ===\n")

	cookies, _ := config.LoadCookies()
	client, _ := api.NewClient(cookies, api.WithModel(models.Model30Pro), api.WithAutoRefresh(false))
	defer client.Close()
	client.Init()

	// Prompt grande como o CLI faria
	userPrompt := strings.Repeat("Contexto de teste. ", 6000) + "\n\nResponda apenas: OK"
	fmt.Printf("Prompt size: %.1f KB (threshold: %.1f KB)\n", float64(len(userPrompt))/1024, float64(api.LargePromptThreshold)/1024)

	// Upload
	fmt.Printf("[%s] Uploading...\n", time.Now().Format("15:04:05"))
	uploadedFile, err := client.UploadText(userPrompt, "prompt.md")
	if err != nil {
		fmt.Printf("Upload failed: %v\n", err)
		return
	}
	fmt.Printf("[%s] Upload OK: %s\n\n", time.Now().Format("15:04:05"), uploadedFile.ResourceID)

	// Generate com "."
	fmt.Printf("[%s] Generating with '.'...\n", time.Now().Format("15:04:05"))
	start := time.Now()

	done := make(chan bool, 1)
	var output *models.ModelOutput
	var genErr error

	go func() {
		output, genErr = client.GenerateContent(".", &api.GenerateOptions{
			Images: []*api.UploadedFile{uploadedFile},
		})
		done <- true
	}()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			fmt.Printf("[%s] Done in %v\n", time.Now().Format("15:04:05"), time.Since(start))
			if genErr != nil {
				fmt.Printf("Error: %v\n", genErr)
			} else {
				fmt.Printf("Response: %s\n", output.Candidates[0].Text)
			}
			return
		case <-ticker.C:
			fmt.Printf("[%s] Waiting... (%v elapsed)\n", time.Now().Format("15:04:05"), time.Since(start))
		case <-time.After(180 * time.Second):
			fmt.Println("TIMEOUT!")
			return
		}
	}
}
