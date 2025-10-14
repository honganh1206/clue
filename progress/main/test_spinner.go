package main

import (
	"fmt"
	"time"

	"github.com/honganh1206/clue/progress"
)

func main() {
	fmt.Println("Testing Spinner functionality...")

	// Test 1: Basic spinner with message
	fmt.Println("\nTest 1: Basic spinner")
	spinner1 := progress.NewSpinner("Loading data")

	for range 20 {
		fmt.Printf("\r%s", spinner1.String())
		time.Sleep(200 * time.Millisecond)
	}
	spinner1.Stop()
	fmt.Printf("\r%s✓ Done\n", spinner1.String())

	// Test 2: Changing message while spinning
	fmt.Println("\nTest 2: Changing message")
	spinner2 := progress.NewSpinner("Processing files")

	messages := []string{
		"Processing files",
		"Validating data",
		"Generating report",
		"Finalizing",
	}

	for _, msg := range messages {
		spinner2.SetMessage(msg)
		for range 10 {
			fmt.Printf("\r%s", spinner2.String())
			time.Sleep(150 * time.Millisecond)
		}
	}
	spinner2.Stop()
	fmt.Printf("\r%s✓ Complete\n", spinner2.String())

	// Test 3: Long message
	fmt.Println("\nTest 3: Long message")
	spinner3 := progress.NewSpinner("This is a very long message that might need to be handled properly by the spinner implementation")

	for range 15 {
		fmt.Printf("\r%s", spinner3.String())
		time.Sleep(100 * time.Millisecond)
	}
	spinner3.Stop()
	fmt.Printf("\r%s✓ Finished\n", spinner3.String())

	// Test 4: Empty message
	fmt.Println("\nTest 4: Empty message")
	spinner4 := progress.NewSpinner("")

	for range 10 {
		fmt.Printf("\r%s", spinner4.String())
		time.Sleep(100 * time.Millisecond)
	}
	spinner4.Stop()
	fmt.Printf("\r%s✓ Done\n", spinner4.String())

	fmt.Println("\nAll tests completed!")
}
