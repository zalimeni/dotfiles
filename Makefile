BINARY  := sp2md
PKG     := ./...
GO      := go

.PHONY: build test lint clean

build:
	$(GO) build -o $(BINARY) .

test:
	$(GO) test -v -race $(PKG)

lint:
	golangci-lint run $(PKG)

clean:
	rm -f $(BINARY)
