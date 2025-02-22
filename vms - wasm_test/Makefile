all:
	@GOOS=wasip1 GOARCH=wasm go build -o example.wasm example/example.go
	@xxd example.wasm | head -n 1
	@go run main.go
