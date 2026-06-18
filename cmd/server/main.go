package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"uno-api/internal/api"
	"uno-api/internal/game"
	"uno-api/internal/replication"
)

func main() {
	port := flag.Int("port", 8080, "Porta do servidor")
	id := flag.String("id", "srv-01", "Identificador do servidor")
	flag.Parse()

	if envPort := os.Getenv("PORT"); envPort != "" {
		fmt.Sscanf(envPort, "%d", port)
	}
	if envID := os.Getenv("SERVER_ID"); envID != "" {
		*id = envID
	}

	peerList := os.Getenv("PEERS")
	var peerURLs []string
	if peerList != "" {
		for _, u := range strings.Split(peerList, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				peerURLs = append(peerURLs, u)
			}
		}
	}

	pm := game.NewPartidaManager()
	endereco := fmt.Sprintf("http://localhost:%d", *port)

	cs := replication.NewClusterState(*id, endereco, peerURLs, pm)
	cs.Start()

	h := api.NewHandler(pm, cs, *id, endereco)

	r := gin.Default()
	r.Use(api.RecoveryMiddleware())
	r.Use(api.CORSMiddleware())
	api.SetupRoutes(r, h)

	log.Printf("Servidor %s iniciado em %s (cluster: %d peers, lider: %v)",
		*id, endereco, len(peerURLs), cs.IsLeader())

	if err := r.Run(fmt.Sprintf(":%d", *port)); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
