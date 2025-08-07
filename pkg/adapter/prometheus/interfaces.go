package prometheus

import "net/http"

type roundTripper interface {
	http.RoundTripper
}
