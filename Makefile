run/new:
	go run ./main.go
run/latest:
	go run ./main.go -n=false
run/gemini:
	go run ./main.go --provider=google
debug/cli:
	go run ./main.go --tui=false
serve:
	go run ./main.go serve
list/models:
	go run ./main.go list
list/conversations:
	go run ./main.go conversation -l
build:
	$(eval VERSION := $(shell cat VERSION))
	go build -ldflags="-s -X 'github.com/honganh1206/clue/cmd.Version=$(VERSION)'" -o bin/clue main.go
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out
benchmark:
	go test ./... -bench=. -benchmem
