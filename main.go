package main

import (
	"fmt"
	"knowledge-base-assistant/config"
	"os"

	"gopkg.in/yaml.v3"
)

type T struct {
	A int
	B int
}

func main() {

	data, err := os.ReadFile("config/langchain.yml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	var lconfig config.LangChain
	err = yaml.Unmarshal(data, &lconfig)
	if err != nil {
		fmt.Println("Error unmarshalling YAML file", err)
	}

	fmt.Println(lconfig)
}
