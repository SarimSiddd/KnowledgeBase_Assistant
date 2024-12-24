package main

import (
	"context"
	"fmt"
	"knowledge-base-assistant/config"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/tmc/langchaingo/llms"
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

	prompt := "What is the capital of France?"
	completion, err := llms.GenerateFromSinglePrompt(context.Background(), llm, prompt)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(completion)

}
