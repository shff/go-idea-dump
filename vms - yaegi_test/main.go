package main

import (
	"fmt"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func main() {
	// Create a new interpreter
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)

	// Define Go code as a string
	code := `
		package main
		import "fmt"

		func Hello() string {
			return Printf("Hello, %s!", "Yaegi")
		}
	`

	// Evaluate the Go code
	_, err := i.Eval(code)
	if err != nil {
		panic(err)
	}

	// Call the interpreted function
	value, err := i.Eval("main.Hello()")
	if err != nil {
		panic(err)
	}

	// Print the result
	fmt.Println(value)
}
