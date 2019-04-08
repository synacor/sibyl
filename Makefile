IMG ?= synacor/sibyl

bin/sibyl: test
	go get ./...
	go get github.com/GeertJohan/go.rice/rice/...
	go build -o bin/sibyl
	rice append --exec bin/sibyl

install: bin/sibyl
	install bin/sibyl /usr/local/bin/sibyl
	sudo setcap cap_net_bind_service=+ep /usr/local/bin/sibyl

docker-build: test
	docker build -t $(IMG) .

clean:
	rm bin/*

test:
	go test -coverprofile=coverage.out ./...

coverage: test
	go tool cover -html=coverage.out

.PHONY: sibyl install docker-build clean test coverage
