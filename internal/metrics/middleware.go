package metrics

import (
	"net/http"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wResponseWriter := websocketResponseWriter{ResponseWriter: w}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(&wResponseWriter, r)
	})
}
