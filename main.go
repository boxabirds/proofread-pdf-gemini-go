package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	genai "github.com/google/generative-ai-go/genai"
	"github.com/tmc/langchaingo/documentloaders"
	"google.golang.org/api/option"
)

type PageInfo struct {
	PageNumber int
	Content    string
}

func printResponse(resp *genai.GenerateContentResponse) {
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for i, part := range cand.Content.Parts {
				fmt.Printf("part %d: %s\n", i, part)
			}
		}
	}
	fmt.Println("---")
}

func readPdf(path string) ([]PageInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening PDF file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %v", err)
	}
	fileSize := fileInfo.Size()

	loader := documentloaders.NewPDF(file, fileSize)
	ctx := context.Background()
	documents, err := loader.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading PDF: %v", err)
	}

	var pageContents []PageInfo

	for _, doc := range documents {
		pageNumber := doc.Metadata["page"].(int)
		pageContent := doc.PageContent

		pageInfo := PageInfo{
			PageNumber: pageNumber,
			Content:    pageContent,
		}

		pageContents = append(pageContents, pageInfo)
	}

	return pageContents, nil
}

func main() {
	const DEFAULT_GEMINI_MODEL = "gemini-1.5-flash-latest"
	const SYSTEM_PROMPT = `You are a proofreading assistant for a formal, scientific document. 
		Review the following excerpt and provide feedback on
		spelling, punctuation, grammar, verbosity and tone of voice. Suggest fixes where necessary in markdown format,
		quoting the original text and issue in bold`

	inputPdf := flag.String("input-pdf", "", "Path to the input PDF file")
	//modelString := flag.String("model", DEFAULT_GEMINI_MODEL, "Model to use for the API")
	flag.Parse()

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	defer client.Close()

	pageInfos, err := readPdf(*inputPdf)
	if err != nil {
		log.Fatalf("Error reading PDF: %v\n", err)
	}

	for _, pageInfo := range pageInfos {
		pageString := fmt.Sprintf("Page: %d\n%s", pageInfo.PageNumber, pageInfo.Content)
		fmt.Println(pageString)
		fmt.Println("---")
	}

	// model := client.GenerativeModel(*modelString)
	// start := time.Now()

	// // Send each page's content to Gemini for proofreading
	// 	resp, err := model.GenerateContent(ctx, genai.Text(systemPrompt+" "+text))
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	printResponse(resp)

	// 	// Example of marking up issues and suggesting fixes (simplified for demonstration)
	// 	// In practice, you would need to implement logic to parse the Gemini response and apply markup
	// 	// This could involve parsing the response text, identifying issues, and applying markdown or HTML tags
	// 	// For simplicity, this example just prints the response without markup
	// 	fmt.Println("Feedback for page", i, ":")
	// 	fmt.Println(resp.Candidates[0].Content.Parts[0]) // Simplified example

	// elapsed := time.Since(start)
	// fmt.Printf("\nTotal Execution Time: %s\n", elapsed)
}
