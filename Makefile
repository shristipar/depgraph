.PHONY: build run tidy clean

build:
	go build -o depgraph ./cmd/

tidy:
	go mod tidy

run-example:
	go run ./cmd/ --url https://github.com/gin-gonic/gin --lang go --format dot

clean:
	rm -f depgraph dependency_graph.*

test:
	go test ./...
