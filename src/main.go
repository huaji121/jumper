package main

import (
	"fmt"
	"os"
)

func main() {
	game, err := NewGame()
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
	defer game.Cleanup()

	if err := game.Run(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
}
