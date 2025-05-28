package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Printing runting args")
	for _, arg := range os.Args[1:] {
		fmt.Println("arg", arg)
	}
}
