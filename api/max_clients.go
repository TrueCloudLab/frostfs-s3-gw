package api

import (
	"net/http"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api/errors"
)

type (
	// MaxClients provides HTTP handler wrapper with the client limit.
	MaxClients interface {
		Handle(http.HandlerFunc) http.HandlerFunc
	}

	maxClients struct {
		pool    chan struct{}
		timeout time.Duration
	}
)

const defaultRequestDeadline = time.Second * 30

// NewMaxClientsMiddleware returns MaxClients interface with handler wrapper based on
// the provided count and the timeout limits.
func NewMaxClientsMiddleware(count int, timeout time.Duration) MaxClients {
	if timeout <= 0 {
		timeout = defaultRequestDeadline
	}

	return &maxClients{
		pool:    make(chan struct{}, count),
		timeout: timeout,
	}
}

// Handler wraps HTTP handler function with logic limiting access to it.
func (m *maxClients) Handle(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.pool == nil {
			f.ServeHTTP(w, r)
			return
		}

		deadline := time.NewTimer(m.timeout)
		defer deadline.Stop()

		select {
		case m.pool <- struct{}{}:
			defer func() { <-m.pool }()
			f.ServeHTTP(w, r)
		case <-deadline.C:
			// Send a http timeout message
			WriteErrorResponse(w, GetReqInfo(r.Context()), errors.GetAPIError(errors.ErrOperationTimedOut))
			return
		case <-r.Context().Done():
			return
		}
	}
}
