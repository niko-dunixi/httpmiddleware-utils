package httpmiddlewareutils

import (
	"log"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func ExampleChain() {
	middlewareChain := Chain(
		exampleTimingMiddleware(),
	)
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi Mom!"))
	}))
	server := http.Server{
		Handler: middlewareChain(mux),
	}
	// error handling omitted for brevity
	defer server.Close()
	_ = server.ListenAndServe()
}

func exampleTimingMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			defer func() {
				end := time.Now()
				log.Printf("request took %s time", end.Sub(start))
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func TestCombineMiddleware(t *testing.T) {
	request, err := http.NewRequest("GET", "http://www.example.com", nil)
	t.Run("empty middleware", func(t *testing.T) {
		// Setup
		Ω := NewWithT(t)
		expectedHeader := http.StatusOK
		invocationCount := atomic.Int32{}
		writer := httptest.NewRecorder()
		Ω.Expect(err).ShouldNot(HaveOccurred(), "couldn't initialize test http.Request")
		middleware := Chain()
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Ω.Expect(r.Method).To(Equal("GET"))
			Ω.Expect(r.URL.Scheme).Should(Equal("http"))
			Ω.Expect(r.URL.Host).Should(Equal("www.example.com"))
			w.WriteHeader(expectedHeader)
			invocationCount.Add(1)
		}))
		// Action
		handler.ServeHTTP(writer, request)
		// Assertions
		Ω.Expect(writer.Result().StatusCode).Should(Equal(expectedHeader), "invalid status code was returned")
		Ω.Expect(int(invocationCount.Load())).To(Equal(1), "incorrect number of invocations")
	})
	t.Run("singleton middleware", func(t *testing.T) {
		// Setup
		Ω := NewWithT(t)
		expectedHeader := http.StatusOK
		invocationCount := atomic.Int32{}
		writer := httptest.NewRecorder()
		Ω.Expect(err).ShouldNot(HaveOccurred(), "couldn't initialize test http.Request")
		middleware := Chain(func(next http.Handler) http.Handler {
			invocationCount.Add(1)
			return next
		})
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Ω.Expect(r.Method).To(Equal("GET"))
			Ω.Expect(r.URL.Scheme).Should(Equal("http"))
			Ω.Expect(r.URL.Host).Should(Equal("www.example.com"))
			w.WriteHeader(expectedHeader)
			invocationCount.Add(1)
		}))
		// Action
		handler.ServeHTTP(writer, request)
		// Assertions
		Ω.Expect(writer.Result().StatusCode).Should(Equal(expectedHeader), "invalid status code was returned")
		Ω.Expect(int(invocationCount.Load())).To(Equal(2), "incorrect number of invocations")
	})
	t.Run("3 middleware", func(t *testing.T) {
		// Setup
		Ω := NewWithT(t)
		expectedHeader := http.StatusOK
		invocationCount := atomic.Int32{}
		writer := httptest.NewRecorder()
		Ω.Expect(err).ShouldNot(HaveOccurred(), "couldn't initialize test http.Request")
		var timeClientRequest time.Time
		var timeMiddlewareA time.Time
		var timeMiddlewareB time.Time
		var timeMiddlewareC time.Time
		var timeHandler time.Time
		middleware := Chain(
			testTimeMiddleware(Ω, "middleware a", &invocationCount, &timeMiddlewareA),
			testTimeMiddleware(Ω, "middleware b", &invocationCount, &timeMiddlewareB),
			testTimeMiddleware(Ω, "middleware c", &invocationCount, &timeMiddlewareC),
		)
		finalHandler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			timeHandler = time.Now().UTC()
			w.WriteHeader(expectedHeader)
			invocationCount.Add(1)
		}))
		// Action
		timeClientRequest = time.Now().UTC()
		finalHandler.ServeHTTP(writer, request)
		// Assertions
		Ω.Expect(writer.Result().StatusCode).Should(Equal(expectedHeader), "invalid status code was returned")
		Ω.Expect(int(invocationCount.Load())).To(Equal(4), "incorrect number of invocations")
		Ω.Expect(timeClientRequest.Before(timeMiddlewareA)).Should(BeTrueBecause("[timeClientRequest,timeMiddlewareA] middleware was not executed when expected"))
		Ω.Expect(timeMiddlewareA.Before(timeMiddlewareB)).Should(BeTrueBecause("[timeMiddlewareA,timeMiddlewareB] middleware was not executed when expected"))
		Ω.Expect(timeMiddlewareB.Before(timeMiddlewareC)).Should(BeTrueBecause("[timeMiddlewareB,timeMiddlewareC] middleware was not executed when expected"))
		Ω.Expect(timeMiddlewareC.Before(timeHandler)).Should(BeTrueBecause("[timeMiddlewareC,timeHandler] middleware was not executed when expected"))
	})
}

func testTimeMiddleware(Ω Gomega, name string, invocationCounter *atomic.Int32, t *time.Time) Middleware {
	return Middleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Ω.Expect(r.Method).To(Equal("GET"))
			Ω.Expect(r.URL.Scheme).Should(Equal("http"))
			Ω.Expect(r.URL.Host).Should(Equal("www.example.com"))
			time.Sleep(time.Millisecond * 5)
			*t = time.Now()
			log.Printf("executing %s at %s", name, *t)
			invocationCounter.Add(1)
			next.ServeHTTP(w, r)
		})
	})
}
