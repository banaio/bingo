package langserver

import (
	"errors"
	"log"
	"sync"
)

var (
	errLanguageServerIsShuttingDown  = errors.New("language server is shutting down")
	errLanguageServerAlreadyShutdown = errors.New("language server received a shutdown request after it was already shut down")
)

// HandlerCommon contains functionality that both the build and lang
// handlers need. They do NOT share the memory of this HandlerCommon
// struct; it is just common functionality. (Unlike HandlerCommon,
// HandlerShared is shared in-memory.)
type HandlerCommon struct {
	mu       sync.Mutex // guards all fields
	shutdown bool
}

// ShutDown marks this server as being shut down and causes all future calls to checkReady to return an error.
func (h *HandlerCommon) ShutDown() {
	h.mu.Lock()
	if h.shutdown {
		log.Println(errLanguageServerAlreadyShutdown)
	}
	h.shutdown = true
	h.mu.Unlock()
}

// CheckReady returns an error if the handler has been shut
// down.
func (h *HandlerCommon) CheckReady() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.shutdown {
		return errLanguageServerIsShuttingDown
	}
	return nil
}
