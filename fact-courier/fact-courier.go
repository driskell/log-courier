package main

import "fmt"

func main() {
	collector, err := NewMuninCollector()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}

	collector.Collect()
}
