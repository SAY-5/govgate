package main

import "fmt"

// benchRegress is wired in v2 to drive the throughput regression gate. The
// command is registered now so the CLI surface is stable.
func benchRegress(_ []string) error {
	fmt.Println("benchregress: gate is configured in v2")
	return nil
}
