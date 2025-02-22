package main

import (
	"io"
	"net"
	"net/http"
	"sync"
)

var tunnels = sync.Map{}

func main() {
	var domain string = "example.com"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tunnel, ok := tunnels.Load(r.Host)
		if !ok {
			http.Error(w, "Tunnel not found", http.StatusNotFound)
			return
		}

		clientConn := tunnel.(net.Conn)
		err := r.Write(clientConn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Send the response back to the client
		io.Copy(w, clientConn)
	})
	go http.ListenAndServe(":80", nil)

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	for {
		conn, _ := listener.Accept()
		go func() {
			tunnels.Store(domain, conn)
			defer tunnels.Delete(domain)

			io.Copy(conn, conn)
		}()
	}
}
