run/new:
	go run ./main.go
run/latest:
	go run ./main.go -n=false
run/gemini:
	go run ./main.go --provider=google
serve:
	go run ./main.go serve
list/models:
	go run ./main.go list
list/conversations:
	go run ./main.go conversation -l
build:
	go build -ldflags='-s' -o bin/clue main.go
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out
benchmark:
	go test ./... -bench=. -benchmem
