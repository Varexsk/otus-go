package main

import (
	"fmt"

	"golang.org/x/example/hello/reverse"
)

func main() {
	hello := "Hello, OTUS!"

	fmt.Println(reverse.String(hello))
}
