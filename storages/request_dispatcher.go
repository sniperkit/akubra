package storages

import (
	"net/http"

	"github.com/allegro/akubra/log"
	"github.com/allegro/akubra/storages/backend"
)

type dispatcher interface {
	Dispatch(request *http.Request) (*http.Response, error)
}

// RequestDispatcher passes requests and responses to matching replicators and response pickers
type RequestDispatcher struct {
	Backends                  []*backend.Backend
	syncLog                   *SyncSender
	pickClientFactory         func(*http.Request) func([]*backend.Backend) client
	pickResponsePickerFactory func(*http.Request) func(<-chan BackendResponse) picker
}

// NewRequestDispatcher creates RequestDispatcher instance
func NewRequestDispatcher(backends []*backend.Backend, syncLog *SyncSender) *RequestDispatcher {

	return &RequestDispatcher{
		Backends:                  backends,
		syncLog:                   syncLog,
		pickResponsePickerFactory: defaultPickResponsePickerFactory,
		pickClientFactory:         defaultReplicationClientFactory,
	}
}

// Dispatch creates and calls replicators and response pickers
func (rd *RequestDispatcher) Dispatch(request *http.Request) (*http.Response, error) {
	clientFactory := rd.pickClientFactory(request)
	cli := clientFactory(rd.Backends)
	respChan := cli.Do(request)
	pickerFactory := rd.pickResponsePickerFactory(request)
	pickr := pickerFactory(respChan)
	go pickr.SendSyncLog(rd.syncLog)
	return pickr.Pick()
}

type picker interface {
	Pick() (*http.Response, error)
	SendSyncLog(*SyncSender)
}

type client interface {
	Do(*http.Request) <-chan BackendResponse
	Cancel() error
}

var defaultReplicationClientFactory = func(request *http.Request) func([]*backend.Backend) client {
	if isMultiPartUploadRequest(request) {
		return newMultiPartRoundTripper
	}
	return newReplicationClient
}

var defaultPickResponsePickerFactory = func(request *http.Request) func(<-chan BackendResponse) picker {
	log.Println("is bucket, is put", request.URL.Path, request.Method)
	if isBucketPath(request.URL.Path) && (request.Method != http.MethodPut) {
		log.Println("delete")
		return newResponseHandler
	}
	if isBucketPath(request.URL.Path) && (request.Method == http.MethodPut) {
		log.Println("delete")
		return newDeleteResponsePicker
	}
	if request.Method == http.MethodDelete {
		log.Println("delete delete")
		return newDeleteResponsePicker
	}
	log.Println("std")
	return newObjectResponsePicker
}