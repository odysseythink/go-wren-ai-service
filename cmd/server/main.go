package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/odysseythink/go-wren-ai-service/internal/config"
	"github.com/odysseythink/go-wren-ai-service/internal/handler"
	"github.com/odysseythink/go-wren-ai-service/internal/service"
)

func main() {
	cfg := config.Load()

	components, err := service.GenerateComponents(cfg)
	if err != nil {
		log.Fatalf("failed to generate pipeline components: %v", err)
	}

	container := service.NewContainer(components, cfg)
	router := handler.NewRouter(container)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("go-wren-ai-service starting on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
