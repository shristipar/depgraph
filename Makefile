.PHONY: build build-mcp run tidy clean

build:
	go build -o depgraph ./cmd/

build-mcp:
	go build -o depgraph-mcp ./cmd/depgraph-mcp/

tidy:
	go mod tidy

run-example:
	go run ./cmd/ --url https://github.com/gin-gonic/gin --lang go --format dot

clean:
	rm -f depgraph dependency_graph.*

test:
	go test ./...
