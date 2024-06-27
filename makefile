SRC := main.go

default: run

build:
	go build

run:
	go run $(SRC)
	dot -T png -O graph.dot

auto:
	autocmd -v -t makefile -t '.*\.go' -- make
