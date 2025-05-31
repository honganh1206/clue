run:
	go run ./main.go
list/models:
	go run ./main.go list
list/conversations:
	go run ./main.go conversation list
build:
	go build -ldflags='-s' -o bin/clue main.go
