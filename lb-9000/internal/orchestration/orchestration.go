package orchestration

import (
	"lb-9000/lb-9000/internal/backend"
	"lb-9000/lb-9000/internal/store"
	"net/http"
)

type Orchestration interface {
	// StartObserver starts the observer that watches for changes in the orchestrator
	// it will be run in a separate goroutine
	StartObserver(store store.Store)

	// DirectRequest directs the request to the correct backend
	DirectRequest(request *http.Request, backend *backend.Backend)

	// GetBackendNameFromResponse gets the name of the backend from the response
	GetBackendNameFromResponse(response *http.Response) (string, error)
}
