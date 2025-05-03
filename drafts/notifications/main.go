package main

import (
	"log"
	"os/exec"
)

func main() {
	// Title: "Hello, world!"
	// Body:  "This is a test notification"
	cmd := exec.Command("notify-send", "Hello, world!", "This is a test notification")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to send notification: %v", err)
	}
}
