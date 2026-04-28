package main

import (
	"flag"
	"log"
	"strings"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/httpapi"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to runtime config yaml")
	addr := flag.String("addr", ":8080", "HTTP API listen address")
	debug := flag.Bool("debug", false, "write detailed run request/error traces for HTTP requests")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := httpapi.Serve(cfg, strings.TrimSpace(*addr), *debug); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
