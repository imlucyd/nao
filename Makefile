.PHONY: build
build:
	@go build  -o ./build/nao ./cmd/nao/

clean:
	@rm ./build/*

run:
	@./build/nao

install:
	@go install ./cmd/nao/

uninstall:
	@rm -f ~/go/bin/nao

style: 
	@gofumpt -w ./..