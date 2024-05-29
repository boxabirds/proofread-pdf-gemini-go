package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

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
	const PROOFREADER_SYSTEM_PROMPT = `You are a proofreading assistant for a formal, scientific document. 
		Review the following excerpt and provide feedback on
		spelling, punctuation, grammar, verbosity and tone of voice. Suggest fixes where necessary in markdown format,
		quoting the original text and issue in bold`

	const PDF_TEXT_REFORMATTER_PROMPT = `You are a PDF postprocessor, specialising in fixing a number of problems that happen with raw PDF text extraction. Take the input and produce fixed output text only, with no additional narrative or escapes. Examples and their remedies are below.
	Examples of missing spacing and line breaks:
	Input: 1Section 4Near-Term Responses in a Changing Climate
	Output: 1 Section 4 Near-Term Responses in Changing Climate
	Input: 43Current Status and TrendsSection 2Increased concentrations of GHGs in the atmosphereIncreased emissions of greenhouse gases (GHGs)b)a)c)
	Output: 43\nCurrent Status and Trends\nSection 2\nIncreased concentrations of GHGs in the atmosphere\nIncreased emissions of greenhouse gases (GHGs)b)a)c)
	Input: Observed changeassessment Human contributionassessment Main driverMain driver 1979 - mid-1990sSouthern HemisphereMain driverMain driverMain driverLimited evidence & medium agreement Main driverMain driverMain driverMain driverChange in indicatorWarming of global mean surface air temperature since 1850-1900Warming of the troposphere since 1979Cooling of the lower stratosphere since the mid-20th centuryLarge-scale precipitation and upper troposphere humidity changes since 1979Expansion of the zonal mean Hadley Circulation since the 1980sOcean heat content increase since the 1970sSalinity changes since the mid-20th centuryGlobal mean sea level rise since 1970Arctic sea ice loss since 1979Reduction in Northern Hemisphere springtime snow cover since 1950Greenland ice sheet mass loss since 1990sAntarctic ice sheet mass loss since 1990sRetreat of glaciersIncreased amplitude of the seasonal cycle ofatmospheric CO2 since the early 1960sAcidiﬁcation of the global surface oceanMean surface air temperature over land(about 40% larger than global mean warming)Warming of the global climate system since preindustrial timesmediumconﬁdencelikely / highconﬁdencevery likelyextremelylikelyvirtuallycertainfactAtmosphere and water cycleOceanCryosphereCarbon cycleLand climateSynthesisKe
	Output: Observed change assessment\nHuman contribution assessment\nMain driver\nMain driver 1979 - mid-1990s\nSouthern Hemisphere\nMain driver\nMain driver\nMain driver\nLimited evidence & medium agreement\nMain driver\nMain driver\nMain driver\nMain driver\nChange in indicator\nWarming of global mean surface air temperature since 1850-1900\nWarming of the troposphere since 1979\nCooling of the lower stratosphere since the mid-20th century\nLarge-scale precipitation and upper troposphere humidity changes since 1979\nExpansion of the zonal mean Hadley Circulation since the 1980s\nOcean heat content increase since the 1970s\nSalinity changes since the mid-20th century\nGlobal mean sea level rise since 1970\nArctic sea ice loss since 1979\nReduction in Northern Hemisphere springtime snow cover since 1950Greenland ice sheet mass loss since 1990sAntarctic ice sheet mass loss since 1990sRetreat of glaciersIncreased amplitude of the seasonal cycle ofatmospheric CO2 since the early 1960sAcidiﬁcation of the global surface oceanMean surface air temperature over land(about 40% larger than global mean warming)Warming of the global climate system since preindustrial timesmediumconﬁdencelikely / highconﬁdencevery likelyextremelylikelyvirtuallycertainfactAtmosphere and water cycleOceanCryosphereCarbon cycleLand climateSynthesisKe`

	inputPdf := flag.String("input-pdf", "", "Path to the input PDF file")
	modelString := flag.String("model", DEFAULT_GEMINI_MODEL, "Model to use for the API")
	flag.Parse()

	//ctx := context.Background()
	timeoutDuration := 120 * time.Second // Adjust the timeout duration as needed
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	defer client.Close()
	model := client.GenerativeModel(*modelString)
	model.SetTemperature(0.0)

	pageInfos, err := readPdf(*inputPdf)
	if err != nil {
		log.Fatalf("Error reading PDF: %v\n", err)
	}

	testPages := []int{30}
	for _, pageNum := range testPages {
		pageInfo := pageInfos[pageNum-1]
		pageString := fmt.Sprintf("Page: %d\n%s", pageInfo.PageNumber, pageInfo.Content)

		fmt.Println("=== Raw page ===")
		fmt.Println(pageString)

		resp, err := model.GenerateContent(ctx, genai.Text(PDF_TEXT_REFORMATTER_PROMPT+" "+pageString))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("=== Tidied up page ===")
		tidiedPart := resp.Candidates[0].Content.Parts[0]
		tidied, ok := tidiedPart.(genai.Text)
		if !ok {
			// Handle the case where the type assertion fails
			log.Fatalf("Failed to convert %v to genai.Text", tidiedPart)
		}
		fmt.Printf("%s\n", tidiedPart)

		start := time.Now()

		proofReadResp, err := model.GenerateContent(ctx, genai.Text(PROOFREADER_SYSTEM_PROMPT+" "+tidied))

		elapsed := time.Since(start)
		fmt.Printf("\nProofreading Execution Time: %s\n", elapsed)
		if err != nil {
			// you might get blocked: candidate: FinishReasonRecitation where the model seems to get stuck on repetititive or redundant text.
			log.Fatal(err)
		}
		proofReadText := proofReadResp.Candidates[0].Content.Parts[0]
		fmt.Println("=== Proofread page ===")
		fmt.Printf("%s\n", proofReadText)

	}

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
