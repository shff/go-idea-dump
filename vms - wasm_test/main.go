package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func main() {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Instantiate WASI, which implements host functions needed for TinyGo to
	// implement `panic`.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Load the WASM file
	wasmBytes, err := os.ReadFile("example.wasm")
	if err != nil {
		panic(err)
	}

	// Instantiate the guest Wasm into the same runtime. It exports the `add`
	// function, implemented in WebAssembly.
	mod, err := r.Instantiate(ctx, wasmBytes)
	if err != nil {
		log.Panicf("failed to instantiate module: %v", err)
	}

	// // Call the `add` function and print the results to the console.
	// add := mod.ExportedFunction("Add")

	for i, f := range mod.ExportedFunctionDefinitions() {
		fmt.Printf("[%s] - [%s]\n", i, f.Name())
	}
	// results, err := add.Call(ctx, 2, 2)
	// if err != nil {
	// 	log.Panicf("failed to call add: %v", err)
	// }

	// fmt.Printf("%d + %d = %d\n", 2, 2, results[0])
}
