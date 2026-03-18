package main

import (
	"log"

	"github.com/alecharmon/screensaver/internal/screensaver"
)

func main() {
	if err := screensaver.Run(); err != nil {
		log.Fatal(err)
	}
}

