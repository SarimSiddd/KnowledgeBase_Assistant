package main

import (
	"bufio"
	"context"
	"fmt"
	"knowledge-base-assistant/config"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	chromago "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/hf"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"gopkg.in/yaml.v3"
)

const (
	knowledgeBasePath = "/Users/sarimsiddiqui/Workspace/KnowledgeBase"
	maxFilesToProcess = 5
)

func init() {
	// Set API keys
	os.Setenv("HUGGINGFACEHUB_API_TOKEN", huggingFaceToken)
	os.Setenv("ANTHROPIC_API_KEY", anthropicToken)
}

func shouldProcessFile(path string, info os.FileInfo) bool {
	// Skip .git directory and hidden files
	if strings.Contains(path, ".git") || strings.HasPrefix(filepath.Base(path), ".") {
		return false
	}

	// Skip directories
	if info.IsDir() {
		return false
	}

	// Process only specific file types
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md":
		return true
	default:
		return false
	}
}

func createStructuredPrompt(context string, user_question string) string {
	prompt := fmt.Sprintf(
		`You are a specialized assistant that MUST follow these rules:
	1. ONLY use information from the provided context
	2. Do NOT use any external knowledge
	3. If information isn't in the context, say "This information is not in the provided context"
	4. Support your answer with direct quotes using "..." from the context

	Context:
	%s

	Question: %s
	
	You are to use the below steps but don't need to include them in your response, just provide the answer:

    Step 1: First state whether the context contains enough information to answer the question.
    Step 2: If yes, provide the answer with direct quotes as evidence.
    Step 3: If no, explicitly state what information is missing.`,
		context, user_question,
	)
	return prompt
}

func retryWithBackoff(operation func() error, maxRetries int, initialWait time.Duration) error {
	var err error
	wait := initialWait
	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		if i < maxRetries-1 {
			log.Printf("Operation failed, attempt %d/%d. Waiting %v before retry...", i+1, maxRetries, wait)
			time.Sleep(wait)
			wait *= 2 // Exponential backoff
		}
	}
	return fmt.Errorf("operation failed after %d retries: %v", maxRetries, err)
}

func main() {
	ctx := context.Background()

	// Load configuration
	data, err := os.ReadFile("config/db.yml")
	if err != nil {
		log.Fatal("Error reading config file:", err)
	}

	var dbconfig config.Database
	err = yaml.Unmarshal(data, &dbconfig)
	if err != nil {
		log.Fatal("Error unmarshalling YAML file:", err)
	}

	// Initialize HuggingFace embeddings
	embedder := hf.NewHuggingFaceEmbeddingFunction(
		huggingFaceToken,
		"sentence-transformers/all-MiniLM-L6-v2",
	)

	// Initialize ChromaDB client
	client, err := chromago.NewClient("http://localhost:8000")
	if err != nil {
		log.Fatal("Failed to create ChromaDB client:", err)
	}

	// Create or get collection
	collection, err := client.CreateCollection(ctx, "knowledge-base", nil, true, embedder, types.L2)
	if err != nil {
		// If collection exists, try to get it
		collection, err = client.GetCollection(ctx, "knowledge-base", embedder)
		if err != nil {
			log.Fatal("Failed to get/create collection:", err)
		}
	}

	// Process files from knowledge base directory
	processedFiles := 0
	skippedFiles := 0
	failedFiles := 0

	err = filepath.Walk(knowledgeBasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if processedFiles >= maxFilesToProcess {
			return filepath.SkipDir
		}

		if !shouldProcessFile(path, info) {
			if !info.IsDir() {
				skippedFiles++
			}
			return nil
		}

		log.Printf("Processing file %d/%d: %s", processedFiles+1, maxFilesToProcess, path)

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Error reading file %s: %v", path, err)
			failedFiles++
			return nil
		}

		if len(string(content)) == 0 {
			log.Printf("Skipping empty file: %s", path)
			skippedFiles++
			return nil
		}

		// Create record set
		recordSet, err := types.NewRecordSet(types.WithEmbeddingFunction(embedder))
		if err != nil {
			log.Printf("Error creating record set for %s: %v", path, err)
			failedFiles++
			return nil
		}

		// Add record to set
		recordSet.WithRecord(
			types.WithID(filepath.Base(path)),
			types.WithDocument(string(content)),
			types.WithMetadata("path", path),
		)

		// Build and validate records with retry logic
		var records []*types.Record
		maxRetries := 10
		for i := 0; i < maxRetries; i++ {
			records, err = recordSet.BuildAndValidate(ctx)
			if err != nil {
				if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "503") {
					waitTime := time.Second * time.Duration(10<<i) // Start with 10 seconds and increase exponentially
					log.Printf("Rate limited, waiting %v before retry %d/%d for %s", waitTime, i+1, maxRetries, path)
					time.Sleep(waitTime)
					continue
				}
				log.Printf("Error building records for %s: %v", path, err)
				failedFiles++
				return nil
			}
			break
		}

		if records == nil {
			log.Printf("Failed to build records for %s after %d retries", path, maxRetries)
			failedFiles++
			return nil
		}

		// Add records to collection
		embeddings := make([]*types.Embedding, len(records))
		metadatas := make([]map[string]interface{}, len(records))
		documents := make([]string, len(records))
		ids := make([]string, len(records))

		for i, record := range records {
			embeddings[i] = &record.Embedding
			metadatas[i] = record.Metadata
			documents[i] = record.Document
			ids[i] = record.ID
		}

		err = retryWithBackoff(func() error {
			_, err := collection.Add(ctx, embeddings, metadatas, documents, ids)
			return err
		}, 5, time.Second*5)

		if err != nil {
			log.Printf("Error adding document %s to collection: %v", path, err)
			failedFiles++
			return nil
		}

		log.Printf("Successfully processed file: %s", path)
		processedFiles++

		if processedFiles >= maxFilesToProcess {
			return filepath.SkipDir
		}

		time.Sleep(time.Second * 5) // Rate limiting between files
		return nil
	})

	if err != nil && err != filepath.SkipDir {
		log.Fatal("Error processing files:", err)
	}

	log.Printf("\nProcessing complete:\n- Processed: %d files\n- Skipped: %d files\n- Failed: %d files\n",
		processedFiles, skippedFiles, failedFiles)

	// Initialize Anthropic LLM
	llm, err := anthropic.New()
	if err != nil {
		log.Fatal("Failed to initialize Anthropic LLM:", err)
	}

	// Interactive search loop
	fmt.Println("\nEnter your question (or 'quit' to exit):")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		question := strings.TrimSpace(scanner.Text())
		if question == "" {
			continue
		}

		if strings.ToLower(question) == "quit" {
			break
		}

		// Search similar documents with retry
		var results *chromago.QueryResults
		err = retryWithBackoff(func() error {
			var err error
			results, err = collection.Query(ctx, []string{question}, 3, nil, nil, nil)
			return err
		}, 5, time.Second*5)

		if err != nil {
			log.Printf("Error querying collection: %v", err)
			continue
		}

		if len(results.Documents) == 0 || len(results.Documents[0]) == 0 {
			fmt.Println("\nNo relevant documents found for your question.")
			continue
		}

		// Create context from search results
		context := ""
		for i, docs := range results.Documents {
			for j, doc := range docs {
				metadata := ""
				if len(results.Metadatas) > i && len(results.Metadatas[i]) > j {
					if path, ok := results.Metadatas[i][j]["path"].(string); ok {
						metadata = fmt.Sprintf(" (Source: %s)", filepath.Base(path))
					}
				}
				context += fmt.Sprintf("Document %d%s:\n%s\n\n", i+1, metadata, doc)
			}
		}

		// Generate response using Anthropic
		prompt := createStructuredPrompt(context, question)
		response, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
		if err != nil {
			log.Printf("Error generating response: %v", err)
			continue
		}

		fmt.Printf("\nResponse: %s\n\n", response)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}
}
