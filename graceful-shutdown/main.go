package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancelCause(context.Background())

	srv := &http.Server{Addr: ":8080"}

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			cancel(err)
		}
	}()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel(errors.New("exit signal received"))
	}()

	<-ctx.Done()

	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()

	err := srv.Shutdown(cleanupCtx)
	if err != nil {
		fmt.Println("Server shutdown error:", err)
	}

	fmt.Println("Exited with:", context.Cause(ctx))
}
