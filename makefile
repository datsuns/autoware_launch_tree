SRC := main.go

default: run

build:
	go build

run:
	go run $(SRC)
	dot -T png -O graph.dot

setup:
	sudo apt install -y golang graphviz

auto:
	autocmd -v -t makefile -t '.*\.go' -- make

.PHONY: default build run setup auto
