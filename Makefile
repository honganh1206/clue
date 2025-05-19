chat:
	go run ./main.go chat
list:
	go run ./main.go list
build:
	go build -ldflags='-s' -o bin/adrift main.go
