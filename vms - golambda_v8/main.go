package main

import (
	"fmt"

	"rogchap.com/v8go"
)

func main() {
	// Initialize V8 isolate
	isolate := v8go.NewIsolate()
	defer isolate.Dispose()

	// define exports.handler so it can be called later
	global := v8go.NewObjectTemplate(isolate)
	global.Set("exports", v8go.NewObjectTemplate(isolate))
	global.Set("print", v8go.NewFunctionTemplate(isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		for _, arg := range info.Args() {
			fmt.Printf("%v ", arg)
		}
		fmt.Println()
		return nil
	}))

	ctx := v8go.NewContext(isolate, global)
	ctx.RunScript("exports.handler = (event, context) => { print('hello from handler!') }", "main.js")
	ctx.RunScript("exports.handler()", "main.js")
}
