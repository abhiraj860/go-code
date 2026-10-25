package main

import (
	"fmt"
	"log"
	"github.com/pkoukk/tiktoken-go"
)

func main() {
	text := "Hello, how are you?"
	tkm, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		log.Fatal(err)
	}
	tokens := tkm.Encode(text, nil, nil)
	fmt.Println(tokens)
	fmt.Println("Token Count:", len(tokens))
}
