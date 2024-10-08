package main

import "fmt"

var list []*string

func main() {
	for _, item := range list {
		fmt.Print(item)
	}
}
