.PHONY: build build-mcp build-verify run tidy clean

build:
	go build -o depgraph ./cmd/

build-mcp:
	go build -o depgraph-mcp ./cmd/depgraph-mcp/

build-verify:
	go build -o depgraph-verify ./cmd/verify/

tidy:
	go mod tidy

run-example:
	go run ./cmd/ --url https://github.com/gin-gonic/gin --lang go --format dot

clean:
	rm -f depgraph dependency_graph.*

test:
	go test ./...
