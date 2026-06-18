EXE=lacrima

build:
	go build -o $(EXE) ./cmd/lacrima

clean:
	rm -f $(EXE)