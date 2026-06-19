package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"uno-api/internal/client"
)

func main() {
	serverList := flag.String("server", "http://localhost:8080",
		"URL do(s) servidor(es). Para failover, informe múltiplos separados por virgula "+
			"(ex: http://localhost:8080,http://localhost:8081). O cliente sempre tenta o primeiro "+
			"(lider); se nao responder, tenta os demais (replicas).")
	flag.Parse()

	if envURL := os.Getenv("SERVER_URL"); envURL != "" {
		*serverList = envURL
	}

	var urls []string
	for _, u := range strings.Split(*serverList, ",") {
		u = strings.TrimSpace(u)
		if u != "" {
			urls = append(urls, u)
		}
	}

	api := client.NewAPIClient(urls...)
	fmt.Printf("Cliente conectado. Servidores conhecidos: %v\n", urls)
	term := client.NewTerminal(api)
	term.Run()
}
