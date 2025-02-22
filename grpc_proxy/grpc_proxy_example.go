package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// HTTP to gRPC proxy
func main() {
	http.HandleFunc("/", handleGRPCProxy)
	log.Println("Starting gRPC-HTTP proxy on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleGRPCProxy(w http.ResponseWriter, r *http.Request) {
	// Extract the gRPC method from the URL (e.g., /package.Service/Method)
	method := strings.TrimPrefix(r.URL.Path, "/")

	// Establish a connection to the gRPC backend
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		http.Error(w, "Failed to connect to gRPC server: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Create a gRPC client for the intercepted method
	client := grpc.NewClientConn(conn)
	ctx := metadata.NewOutgoingContext(r.Context(), extractMetadataFromHeaders(r))

	// Prepare the gRPC request
	stream, err := client.NewStream(ctx, &grpc.StreamDesc{}, method)
	if err != nil {
		http.Error(w, "Failed to create gRPC stream: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Forward HTTP request body to gRPC
	go func() {
		io.Copy(stream, r.Body)
		stream.CloseSend()
	}()

	// Receive gRPC response and stream it back to HTTP client
	for {
		resp, err := stream.RecvMsg()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "Error reading from gRPC stream: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Write gRPC response as HTTP response
		w.Header().Set("Content-Type", "application/grpc")
		w.Write(resp.([]byte))
	}
}

func extractMetadataFromHeaders(r *http.Request) metadata.MD {
	md := metadata.MD{}
	for k, v := range r.Header {
		md[k] = v
	}
	return md
}
