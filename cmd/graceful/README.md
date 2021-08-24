# Graceful shutdown

On gcp cloud run we have the opportunity to gracefully shutdown our application in case gcp decides to scale down our
service. With golang we can capture the **SIGTERM** signal that google will send us to and use that signal to gracefully
shutdown our http server. You can read more about
that [here](https://cloud.google.com/blog/topics/developers-practitioners/graceful-shutdowns-cloud-run-deep-dive)

Let us take a look at the go code for handling this.

## Signal Handling

First we create a base context, this is the context that we will use for application level dependencies (db connection
as an example)

```go
// create our base context to work with
ctx, cancelFunc := context.WithCancel(context.Background())
defer cancelFunc()
```

Now we setup our http server with a base context extraction function

```go
httpServer := &http.Server{
    Addr:    ":" + port,
    Handler: mux,
    // we register our base context function, this will be the first piece of context added to all incoming calls
    // if this context where to be cancelled, it would cancel all subsequent context driven functions therefore to
    // allow for a clean and mostly graceful disconnect
    BaseContext: func (listener net.Listener) context.Context { return ctx },
}
// upon shutdown cancel our base context
httpServer.RegisterOnShutdown(cancelFunc)
```

This will go ahead and allow us to capture one of two signals, os.Interrupt (SIGINT) for local development, and SIGTERM
for when gcp wants to shut down an instance of our service.

```go
// setup our shutdown signal
shutdown := make(chan os.Signal, 1)

signal.Notify(
    shutdown,
    os.Interrupt,    // Capture ctrl + c events (SIGINT)
    syscall.SIGTERM, // Capture actual sig term event (kill command).
)
```

The final piece of the puzzle to go ahead and wait for that signal on a separate go routine and attempt a graceful
shutdown. In this example we will give it 9 seconds to cancel all ongoing context driven operations, if it happens to
clock out after 9 seconds, the httpServer.Shutdown call will be forced to error out.

```go

// setup our errgroup is listen for shutdown signal, from there attempt to shutdown our http server and capture any errors during shutdown
g, ctx := errgroup.WithContext(ctx)
g.Go(func () error {
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

```

## Real use cases

With the above code tied together we will showcase how it performs and how it affects the lifecycle of the application
in regards to handling ongoing operations.

What happens if you where not using the request context for an upstream network/context driven call? In this example we
are using a fresh background context. To see what happens we do the following, start server, make curl request
to `http://localhost:8080/noncancelablerequest` and immediately send a ctrl + c event to our application.

```go
mux.HandleFunc("/noncancelablerequest", func (writer http.ResponseWriter, request *http.Request) {
    
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
```

Produces the following result.

```log
2021/08/23 21:29:45 starting server on ":8080"
2021/08/23 21:29:50 starting work
2021/08/23 21:29:51 sig: interrupt - starting shutting down sequence...
2021/08/23 21:30:00 run(): httpServer.Shutdown(): context deadline exceeded
```

Let's fix that to work correctly now.

```go
mux := http.NewServeMux()
mux.HandleFunc("/cancelablerequest", func (writer http.ResponseWriter, request *http.Request) {
    log.Println("starting work")

    // very important we are using the request.Context()
    // since we are using the requests' context which inherits from our context generator function
    // if we SIGTERM the server, this upstream api call will get canceled
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
```

The result of running our http server and then sending a curl request to our endpoint
of `http://localhost:8080/cancelablerequest` and then immediately hitting ctrl + c will go ahead and start the shutdown
sequence and finish gracefully.

```log
2021/08/23 21:26:41 starting server on ":8080"
2021/08/23 21:26:47 starting work
2021/08/23 21:26:49 sig: interrupt - starting shutting down sequence...
2021/08/23 21:26:49 http.DefaultClient.Do: Get "https://httpbin.org/delay/10": context canceled
2021/08/23 21:26:49 server has shutdown gracefully

```


The full reference source code is

```go
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
```
