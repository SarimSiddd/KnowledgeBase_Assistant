package main

import (
	"context"
	"fmt"
	"knowledge-base-assistant/config"
	"os"

	"github.com/jackc/pgx/v5"
	"gopkg.in/yaml.v3"
)

type T struct {
	A int
	B int
}

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
	}

	defer conn.Close(context.Background())

	var res int64
	err = conn.QueryRow(context.Background(), "SELECT 1 AS res").Scan(&res)

	if err != nil {
		fmt.Println("Unable to execute query on db", err)
	}

	fmt.Println("Printing row result: ", res)
}
