package main

import (
	"fmt"
	"os"

	"github.com/faergeek/go-and-chat/internal/client"
	"github.com/faergeek/go-and-chat/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	mode := os.Args[1]

	var address string
	if len(os.Args) == 2 {
		address = "localhost:1234"
	} else {
		address = os.Args[2]
	}

	switch mode {
	case "server":
		err := server.Start(address)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "client":
		err := client.Start(address)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("Usage: %s <server|client> [[host]:port]\n", os.Args[0])
}
