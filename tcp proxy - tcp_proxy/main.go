package main

import (
	"io"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8080") // Listen on port 8080
	if err != nil {
		log.Fatalf("Failed to bind: %v", err)
	}
	defer listener.Close()
	log.Println("Listening on :8080")

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		go handleConnection(clientConn, "destination-address:port")
	}
}

func handleConnection(clientConn net.Conn, backendAddr string) {
	defer clientConn.Close()

	backendConn, err := net.Dial("tcp", backendAddr)
	if err != nil {
		log.Printf("Failed to connect to backend: %v", err)
		return
	}
	defer backendConn.Close()

	// Proxy data between client and backend
	go io.Copy(backendConn, clientConn)
	io.Copy(clientConn, backendConn)
}
