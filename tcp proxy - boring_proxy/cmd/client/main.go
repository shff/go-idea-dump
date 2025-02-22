package client

import (
	"io"
	"net"
)

func main() {
	serverConn, err := net.Dial("tcp", "server:8080")
	if err != nil {
		panic(err)
	}

	listener, err := net.Listen("tcp", ":3000")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		localConn, _ := listener.Accept()
		go func() {
			defer localConn.Close()

			io.Copy(serverConn, localConn)
		}()
	}
}
