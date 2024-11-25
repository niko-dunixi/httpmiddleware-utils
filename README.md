# httpmiddleware-utils

You need it. I need it. It's easy to mess up, but just as easy to test.
Let's codify it and reuse it so we don't have to re-implement middleware
chaining in every project.

## Usage
The meat and potatoes is the `Chain` function, which will take an arbitrary
set of middleware and collapse it into a singular middleware that is more
ergonomic to use.

There is a type definition for convienience and readability, aptly named
`Middleware`, but otherwise there is no stringent requirement as higher
order functions can be used to adapt anything that does not immediately
conform into compatible middleware.

From [example/main.go](./example/main.go)
```golang
// Define your REST paths, with any necessary complexity, as usual
mux := http.NewServeMux()
mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
	nowString := time.Now().Format(time.RFC3339)
	w.Header().Set("Content-Type", "text/html")
	html := fmt.Sprintf(
		`<html>
			<div>Hi Mom!</div>
			<time datetime="%s">%s</time>
		</html>`,
		nowString, nowString,
	)
	w.Write([]byte(html))
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
```

## Usable Middleware

### Panic Recovery

#### `PanicRecoveryMiddleware`
This is a common middleware in most rest frameworks, but is something you need
to roll yourself when using the standard library. The out-of-the box implementation
will recover from a panic, respond to the client with `500 Internal Server Error`
and record the panic via OTEL.

> If OTEL is not initialized this will be a noop without side-effects

#### `PanicRecoveryMiddlewareFunc`
Will recover from the immediate panic, but allows you as a developer to specify
any arbitrary remediation action desired.
