.PHONY: build


build :
	go build -o ./bin/a ./cmd/operatorA
	go build -o ./bin/b ./cmd/operatorB
	go build -o ./bin/chain ./cmd/chain