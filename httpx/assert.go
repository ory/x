package httpx

import (
	"net/http"

	"github.com/urfave/negroni"
)

func GetResponseMeta(w http.ResponseWriter) (status, size int) {
	switch t := w.(type) {
	case interface {
		Status() int
		Written() int64
	}:
		return t.Status(), int(t.Written())
	case negroni.ResponseWriter:
		return t.Status(), t.Size()
	}

	if t, ok := w.(interface {
		Status() int
	}); ok {
		return t.Status(), 0
	}

	return 0, 0
}
