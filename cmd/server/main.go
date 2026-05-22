package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

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

	// Start HTTP server in a background goroutine so the main goroutine can
	// perform the optional force-deploy startup hook (mirrors Python entrypoint.sh).
	done := make(chan struct{})
	srv := &http.Server{Addr: addr, Handler: router}
	go func() {
		defer close(done)
		log.Printf("go-wren-ai-service starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Wait for server to be ready before running startup hooks.
	if err := waitForServer(addr, 60*time.Second); err != nil {
		log.Fatalf("server did not become ready: %v", err)
	}

	// Force-deploy hook: when SHOULD_FORCE_DEPLOY is set and ENGINE is wren_ui,
	// send a deploy(force:true) GraphQL mutation to wren-ui so it re-indexes.
	if cfg.ShouldForceDeploy && cfg.Engine == "wren_ui" {
		if err := forceDeploy(cfg); err != nil {
			log.Printf("force deploy failed: %v", err)
		}
	}

	// Block until the server goroutine exits (e.g., on graceful shutdown).
	<-done
}
