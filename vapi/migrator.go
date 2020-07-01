package vapi

import "net/http"

type Migrator interface{
	Request(r *http.Request)
	Response(r *http.Request)
}
