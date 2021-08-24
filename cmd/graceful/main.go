package main

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("run(): %v", err)
	}
}

func run() error {

	mux := http.NewServeMux()
	mux.HandleFunc("/cancelablerequest", func(writer http.ResponseWriter, request *http.Request) {

		log.Println("starting work")
		req, err := http.NewRequestWithContext(request.Context(), http.MethodGet, "https://httpbin.org/delay/10", nil)
		if err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		_, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("http.DefaultClient.Do: %v", err)
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(writer, "<h1> hello world <h1/>")
	})

	mux.HandleFunc("/noncancelablerequest", func(writer http.ResponseWriter, request *http.Request) {

		log.Println("starting work")
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://httpbin.org/delay/10", nil)
		if err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		_, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("http.DefaultClient.Do: %v", err)
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(writer, "<h1> hello world <h1/>")
	})

	// create our base context to work with
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
		// we register our base context generator, this will be the first piece of context added to all incoming calls
		// if this context where to be cancelled, it would cancel all subsequent context driven functions therefore to
		// allow for a clean and mostly graceful disconnect
		BaseContext: func(listener net.Listener) context.Context { return ctx },
	}

	// upon shutdown cancel our base context
	httpServer.RegisterOnShutdown(cancelFunc)

	// setup our shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(
		shutdown,
		os.Interrupt,    // Capture ctrl + c events (SIGINT)
		syscall.SIGTERM, // Capture actual sig term event (kill command).
	)

	// setup our errgroup is listen for shutdown signal, from there attempt to shutdown our http server and capture any errors during shutdown
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		// on an seperate go routine we will wait and listen for our shutdown events
		o := <-shutdown
		log.Printf("sig: %s - starting shutting down sequence...", o)
		// we need to use a fresh context.Background() because the parent ctx we have in our current scope will be cancelled during the Shutdown method call
		graceFull, cancel := context.WithTimeout(context.Background(), 9*time.Second)
		defer cancel()
		// Shutdown the server with a timeout
		if err := httpServer.Shutdown(graceFull); err != nil {
			return fmt.Errorf("httpServer.Shutdown(): %w", err)
		}
		log.Printf("server has shutdown gracefully")
		return nil
	})
	log.Printf("starting server on %q", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("httpServer.ListenAndServe(): %v", err)
	}
	return g.Wait()
}
