package main

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type T struct {
	A int
	B int
}

func main() {

	var t T
	yaml.Unmarshal([]byte("a: 1\nb: 2"), &t)
	fmt.Println(t)
}
