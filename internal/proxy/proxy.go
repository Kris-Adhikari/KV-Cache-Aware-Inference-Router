package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/Kris-Adhikari/KV-Aware-Inference-Router/internal/router"
)

type ctxKey int

const targetKey ctxKey = 0

// Proxy routes each request to a backend chosen by its Router and reverse-proxies it there.
type Proxy struct {
	router router.Router
	rp     *httputil.ReverseProxy
}

func New(r router.Router) *Proxy {
	p := &Proxy{router: r}
	p.rp = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(pr.In.Context().Value(targetKey).(*url.URL))
			pr.SetXForwarded()
		},
	}
	return p
}

func routingKey(r *http.Request) string {
	return r.Header.Get("X-Session-Id")
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	b := p.router.Route(routingKey(req))
	if b == nil {
		http.Error(w, "no backend available", http.StatusServiceUnavailable)
		return
	}
	target, err := url.Parse(b.URL)
	if err != nil {
		http.Error(w, "invalid backend url", http.StatusInternalServerError)
		return
	}
	p.rp.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), targetKey, target)))
}
