package main

import (
	"flag"
	"log"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/transport/openai"
)

func main() {
	configPath := flag.String("config", "../gateway.config.json", "path to gateway config")
	flag.Parse()

	srv, err := openai.NewServer(*configPath)
	if err != nil {
		log.Fatalf("gateway init failed: %v", err)
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("gateway stopped: %v", err)
	}
}
