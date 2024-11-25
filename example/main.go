package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"

	httpmiddlewareutils "github.com/niko-dunixi/httpmiddleware-utils"
	"github.com/niko-dunixi/httpmiddleware-utils/panicrecovery"
)

func main() {
	// Define your REST paths, with any necessary complexity, as usual
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		responseMap := make(map[string]any)
		responseMap["message"] = "Hi Mom!"
		responseMap["time"] = time.Now()
		if err := json.NewEncoder(w).Encode(responseMap); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("500 Internal Server Error"))
		}
	})
	server := http.Server{
		Handler: httpmiddlewareutils.Chain(
			// Place any other arbitrary middleware here. Any handler that is passed
			// as the final handler parameter will be wrapped by the full chain of
			// middleware.
			panicrecovery.PanicRecoveryMiddleware(),
			timingMiddleware(),
			slowValidationMiddleware(),
		)(mux),
		Addr: net.JoinHostPort("0.0.0.0", "8080"),
	}
	log.Printf("running server: %v", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}

func timingMiddleware() httpmiddlewareutils.Middleware {
	return httpmiddlewareutils.Middleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			defer func() {
				end := time.Now()
				log.Printf("execution took %s to perform", end.Sub(start))
			}()
			next.ServeHTTP(w, r)
		})
	})
}

func slowValidationMiddleware() httpmiddlewareutils.Middleware {
	return httpmiddlewareutils.Middleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("performing slow validation")
			sleepSeconds := time.Duration(rand.Intn(4)+1) * time.Second
			time.Sleep(sleepSeconds)
			next.ServeHTTP(w, r)
		})
	})
}
