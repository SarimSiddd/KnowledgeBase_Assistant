package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"knowledge-base-assistant/config"
	"log"
	"os"
	"path/filepath"

	chroma "github.com/amikos-tech/chroma-go"
	huggingface "github.com/amikos-tech/chroma-go/hf"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/jackc/pgx/v5"
	"github.com/tmc/langchaingo/llms/anthropic"
	"gopkg.in/yaml.v3"
)

func main() {

	data, err := os.ReadFile("config/db.yml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	var dbconfig config.Database
	err = yaml.Unmarshal(data, &dbconfig)
	if err != nil {
		fmt.Println("Error unmarshalling YAML file", err)
	}

	conn, err := pgx.Connect(context.Background(), "user=sarim password=1234 host=localhost port=3399 dbname=vectors")

	if err != nil {
		fmt.Println("Error connection to database", err)
		panic(err)
	}

	defer conn.Close(context.Background())

	var res int64
	err = conn.QueryRow(context.Background(), "SELECT 1 AS res").Scan(&res)

	if err != nil {
		fmt.Println("Unable to execute query on db", err)
	}

	fmt.Println("Printing row result: ", res)

	llm, err := anthropic.New()
	if err != nil {
		log.Fatal(err)
	}

	_ = llm

	dir := "./samples/"

	embedder := huggingface.NewHuggingFaceEmbeddingFunction(os.Getenv("HUGGINGFACEHUB_API_TOKEN"), "sentence-transformers/all-MiniLM-L6-v2")

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Read the file content
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			// Process the file content
			result, err := embedder.EmbedQuery(context.Background(), string(content))

			if err != nil {
				return err
			}

			fmt.Printf("Embedding result for file %s: %v\n", path, result)
		}

		return nil
	})

	result, err := embedder.EmbedQuery(context.Background(), "Hello World")

	if err != nil {
		log.Fatal(err)
	}

	client, err := chroma.NewClient("http://localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	collection, err := client.CreateCollection(context.Background(), "test-collection", map[string]interface{}{"key": "value"}, true, embedder, types.L2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Collection created: ", collection)

	collection_result, err := collection.Add(context.Background(), []*types.Embedding{result}, []map[string]interface{}{}, []string{"Hello World"}, []string{"doc1"})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Collection result: ", collection_result)

}
