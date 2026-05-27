// Command govgate is the entry point for the GovGate register service and CLI.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		fmt.Println("govgate serve: not yet implemented")
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: govgate <serve|assess|benchregress> [flags]")
}
