package serverx

import "net/http"

// DefaultNotFoundHandler is a default handler for handling 404 errors.
var DefaultNotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)

	if r.Header.Get("Content-type") == "application/json" {
		_, _ = w.Write([]byte(`{"error": "The requested route does not exist"}`)) // #nosec
		return
	}

	_, _ = w.Write([]byte(`The request route does not exist`)) // #nosec
})
