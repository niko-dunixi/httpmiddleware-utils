package panicrecovery

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func ExamplePanicRecoveryMiddleware() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		panic("a fatal error occurred")
	})
	middleware := PanicRecoveryMiddleware()
	server := http.Server{
		Handler: middleware(mux),
	}
	// error handling omitted for brevity
	defer server.Close()
	_ = server.ListenAndServe()
}

func ExamplePanicRecoveryMiddlewareFunc() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		panic("a fatal error occurred")
	})
	middleware := PanicRecoveryMiddlewareFunc(func(w http.ResponseWriter, r *http.Request, recoverValue any, stack []byte) {
		log.Printf("something has broken: %s", recoverValue)
	})
	server := http.Server{
		Handler: middleware(mux),
	}
	// error handling omitted for brevity
	defer server.Close()
	_ = server.ListenAndServe()
	// Output:
	// something has broken: a fatal error occurred
}

func TestPanicRecoveryMiddleware(t *testing.T) {
	t.Run("business as usual", func(t *testing.T) {
		// Setup
		otelMiddleware, exporter := OtelTestingMiddleware(t)
		Ω := NewWithT(t)
		writer := httptest.NewRecorder()
		request, err := http.NewRequest("GET", "example.go", nil)
		Ω.Expect(err).ShouldNot(HaveOccurred(), "couldn't initialize test http.Request")
		handler := otelMiddleware(PanicRecoveryMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("business as usual"))
		})))
		// Action
		handler.ServeHTTP(writer, request)
		// Assertions
		Ω.Expect(writer.Result().StatusCode).Should(Equal(http.StatusOK))
		fullResponseBody, err := io.ReadAll(writer.Result().Body)
		Ω.Expect(err).ShouldNot(HaveOccurred(), "parsing response should not fail")
		Ω.Expect(string(fullResponseBody)).Should(Equal("business as usual"))
		spans := exporter.GetSpans()
		Ω.Expect(spans).To(HaveLen(1), "expect span to be exported")
		Ω.Expect(spans[0].Status.Code).To(Equal(codes.Unset), "error shouldn't be recorded in otel span")
	})
	t.Run("vanilla panic", func(t *testing.T) {
		// Setup
		otelMiddleware, exporter := OtelTestingMiddleware(t)
		Ω := NewWithT(t)
		writer := httptest.NewRecorder()
		request, err := http.NewRequest("GET", "example.go", nil)
		Ω.Expect(err).ShouldNot(HaveOccurred(), "couldn't initialize test http.Request")
		handler := otelMiddleware(PanicRecoveryMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("something broke")
		})))
		// Action
		handler.ServeHTTP(writer, request)
		// Assertions
		Ω.Expect(writer.Result().StatusCode).Should(Equal(http.StatusInternalServerError))
		fullResponseBody, err := io.ReadAll(writer.Result().Body)
		Ω.Expect(err).ShouldNot(HaveOccurred(), "parsing response should not fail")
		Ω.Expect(string(fullResponseBody)).Should(Equal("500 Internal Server Error"))
		spans := exporter.GetSpans()
		Ω.Expect(spans).To(HaveLen(1), "expect span to be exported")
		Ω.Expect(spans[0].Status.Code).To(Equal(codes.Error), "error wasn't recorded in otel span")
	})
	t.Run("nil dereference", func(t *testing.T) {
		// Setup
		otelMiddleware, exporter := OtelTestingMiddleware(t)
		Ω := NewWithT(t)
		writer := httptest.NewRecorder()
		request, err := http.NewRequest("GET", "example.go", nil)
		Ω.Expect(err).ShouldNot(HaveOccurred(), "couldn't initialize test http.Request")
		handler := otelMiddleware(PanicRecoveryMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			aNilReference := NilSupplier[string]()
			log.Printf("i will panic because of a nil dereference: %s", *aNilReference)
		})))
		// Action
		handler.ServeHTTP(writer, request)
		// Assertions
		Ω.Expect(writer.Result().StatusCode).Should(Equal(http.StatusInternalServerError))
		fullResponseBody, err := io.ReadAll(writer.Result().Body)
		Ω.Expect(err).ShouldNot(HaveOccurred(), "parsing response should not fail")
		Ω.Expect(string(fullResponseBody)).Should(Equal("500 Internal Server Error"))
		spans := exporter.GetSpans()
		Ω.Expect(spans).To(HaveLen(1), "expect span to be exported")
		Ω.Expect(spans[0].Status.Code).To(Equal(codes.Error), "error wasn't recorded in otel span")
	})
	t.Run("no default behavior when developers bring their own recovery", func(t *testing.T) {
		// Setup
		otelMiddleware, exporter := OtelTestingMiddleware(t)
		Ω := NewWithT(t)
		writer := httptest.NewRecorder()
		request, err := http.NewRequest("GET", "example.go", nil)
		Ω.Expect(err).ShouldNot(HaveOccurred(), "couldn't initialize test http.Request")
		handler := otelMiddleware(PanicRecoveryMiddlewareFunc(func(w http.ResponseWriter, r *http.Request, recoverValue any, stack []byte) {
			// Validate for the testcase, but don't do anything functional here
			Ω.Expect(w).ToNot(BeNil())
			Ω.Expect(r).ToNot(BeNil())
			Ω.Expect(recoverValue).ToNot(BeNil(), "a value of some king should be present")
			Ω.Expect(stack).ToNot(BeEmpty(), "stack should never be empty")
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("something broke")
		})))
		// Action
		handler.ServeHTTP(writer, request)
		// Assertions
		Ω.Expect(writer.Result().StatusCode).Should(Equal(http.StatusOK))
		fullResponseBody, err := io.ReadAll(writer.Result().Body)
		Ω.Expect(err).ShouldNot(HaveOccurred(), "parsing response should not fail")
		Ω.Expect(string(fullResponseBody)).Should(Equal(""))
		spans := exporter.GetSpans()
		Ω.Expect(spans).To(HaveLen(1), "expect span to be exported")
		Ω.Expect(spans[0].Status.Code).To(Equal(codes.Unset), "error shouldn't be recorded in otel span")
	})
}

func OtelTestingMiddleware(t *testing.T) (func(http.Handler) http.Handler, *tracetest.InMemoryExporter) {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})
	t.Cleanup(func() {
		// likely redundant, since this is not utilized across multiple test-cases, but it
		// almost always makes sense to practice memory-hygiene
		exporter.Reset()
	})
	return otelhttp.NewMiddleware("endpoint", otelhttp.WithTracerProvider(tp)), exporter
}

// A naive funtion to defeat the linter for the purposes of testing
// conditions that the linter should normally catch.
func NilSupplier[T any]() *T {
	return nil
}
