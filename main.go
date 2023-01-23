package main

import (
	"fmt"
	"os"
)
import "goreg/pkg/decompiler"

func main() {
	imageName := "quay.io/jeffmaury/sample1"
	if len(os.Args) > 1 {
		imageName = os.Args[1]
	}
	node, err := decompiler.Decompile(imageName)
	if err != nil {
		panic(err)
	}
	fmt.Println(node.Dump())
}
