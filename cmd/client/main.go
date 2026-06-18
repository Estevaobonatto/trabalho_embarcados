package main

import (
	"flag"
	"os"

	"uno-api/internal/client"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "URL do servidor")
	flag.Parse()

	if envURL := os.Getenv("SERVER_URL"); envURL != "" {
		*serverURL = envURL
	}

	api := client.NewAPIClient(*serverURL)
	term := client.NewTerminal(api)
	term.Run()
}
