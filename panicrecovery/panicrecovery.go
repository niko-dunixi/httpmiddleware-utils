package panicrecovery

import (
	"fmt"
	"net/http"
	"runtime/debug"

	httpmiddlewareutils "github.com/niko-dunixi/httpmiddleware-utils"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type panicErr struct {
	RecoveredError error  `json:"recoveredError,omitempty"`
	RecoveredValue any    `json:"recoveredValue,omitempty"`
	Stack          string `json:"stack,omitempty"`
}

func (err panicErr) Error() string {
	if err.RecoveredError != nil {
		return fmt.Sprintf("a panic ocurred with an error: %v", err.RecoveredError)
	} else if err.RecoveredValue != nil {
		return fmt.Sprintf("a panic occured with value: %v", err.RecoveredValue)
	} else {
		return "a panic occurred from indeterminate cause"
	}
}

// Middleware that recovers from a panic by responding to the request
// with:
//   - http.StatusInternalServerError
//   - Response body of "500 Internal Server Error"
//
// Will also record an error within the OTEL span with the stack
// and recovered value from the panic.
func PanicRecoveryMiddleware() httpmiddlewareutils.Middleware {
	return PanicRecoveryMiddlewareFunc(func(w http.ResponseWriter, r *http.Request, recoverValue any, stack []byte) {
		panicErr := panicErr{
			Stack: string(stack),
		}
		if err, ok := recoverValue.(error); ok {
			panicErr.RecoveredError = err
		} else {
			panicErr.RecoveredValue = recoverValue
		}
		span := trace.SpanFromContext(r.Context())
		span.SetStatus(codes.Error, "fatal panic occurred")
		span.RecordError(panicErr)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
	})
}

// Middleware that allows the developer to specify a recovery action to take when recovering from a panic
func PanicRecoveryMiddlewareFunc(recoverAction func(w http.ResponseWriter, r *http.Request, recoverValue any, stack []byte)) httpmiddlewareutils.Middleware {
	return httpmiddlewareutils.Middleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				value := recover()
				if value == nil {
					return
				}
				stack := debug.Stack()
				recoverAction(w, r, recoverAction, stack)
			}()
			next.ServeHTTP(w, r)
		})
	})
}
