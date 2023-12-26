
all: server

server: dummy
	go build -o main main.go

clean:
	rm main

dummy:
