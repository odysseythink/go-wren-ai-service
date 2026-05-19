module github.com/odysseythink/go-wren-ai-service

go 1.26.3

replace github.com/odysseythink/pantheon => /Users/ranwei/workspace/go_work/pantheon

require (
	github.com/caarlos0/env/v11 v11.4.1
	github.com/go-chi/chi/v5 v5.2.5
	github.com/go-chi/cors v1.2.2
	github.com/google/uuid v1.6.0
	github.com/odysseythink/pantheon v0.0.8
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/sashabaranov/go-openai v1.41.2
	golang.org/x/sync v0.20.0
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/odysseythink/mlog v0.2.3 // indirect
