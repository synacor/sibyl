bin/sibyl:
	go build -o bin/sibyl
	rice append --exec bin/sibyl

install: bin/sibyl
	install bin/sibyl /usr/local/bin/sibyl
	sudo setcap cap_net_bind_service=+ep /usr/local/bin/sibyl

clean:
	rm bin/*

.PHONY: sibyl
