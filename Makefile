IMG ?= synacor/sibyl

bin/sibyl:
	go get ./...
	go get github.com/GeertJohan/go.rice/rice/...
	go build -o bin/sibyl
	rice append --exec bin/sibyl

install: bin/sibyl
	install bin/sibyl /usr/local/bin/sibyl
	sudo setcap cap_net_bind_service=+ep /usr/local/bin/sibyl

docker-build:
	docker build -t $(IMG) .

clean:
	rm bin/*

.PHONY: sibyl install docker-build clean
