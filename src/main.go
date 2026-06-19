package main

import (
	"fmt"
	"os"
)

func main() {
	game, err := NewGame()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialise game: %v\n", err)
		os.Exit(1)
	}
	defer game.Cleanup()

	if err := game.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Game loop error: %v\n", err)
		os.Exit(1)
	}
}
