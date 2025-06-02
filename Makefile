run/new:
	go run ./main.go
run/latest:
	go run ./main.go -c=true
list/models:
	go run ./main.go list
list/conversations:
	go run ./main.go -c -l
build:
	go build -ldflags='-s' -o bin/clue main.go
