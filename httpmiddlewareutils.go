package httpmiddlewareutils

import "net/http"

// Convienience type to define middleware
type Middleware func(next http.Handler) http.Handler

// Take a set of middleware and combine them into a singular middleware
// to be executed in the same order they are provided.
func Chain(mw ...Middleware) Middleware {
	return Middleware(func(finalHandler http.Handler) http.Handler {
		for i := len(mw) - 1; i >= 0; i-- {
			finalHandler = mw[i](finalHandler)
		}
		return finalHandler
	})
}
