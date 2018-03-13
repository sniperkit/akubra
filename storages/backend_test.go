package storages

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/allegro/akubra/storages/config"
	"github.com/allegro/akubra/transport"
	transportConfig "github.com/allegro/akubra/transport/config"
	"github.com/allegro/akubra/utils"
	"github.com/stretchr/testify/require"

	"github.com/allegro/akubra/types"
)

func newBackend(backendConfig config.Backend, transports transport.Container) (*Backend, error) {
	return &Backend{Endpoint: *backendConfig.Endpoint.URL, Transports: transports}, nil
}

type testRt struct {
	rt func(*http.Request) (*http.Response, error)
}

func (trt *testRt) RoundTrip(req *http.Request) (*http.Response, error) {
	return trt.rt(req)
}

func TestBackendShouldChangeRequestHost(t *testing.T) {
	host := "someremote.backend:8080"
	netURL, err := url.Parse(fmt.Sprintf("http://%s", host))
	require.NoError(t, err)

	hostURL := types.YAMLUrl{URL: netURL}
	roundtripper := func(req *http.Request) (*http.Response, error) {
		return &http.Response{Request: req}, nil
	}

	backendConfig := config.Backend{Endpoint: hostURL, Type: "passthrough"}
	b, err := newBackend(backendConfig, prepareTestTransportContainer(roundtripper))
	require.NoError(t, err)

	r, err := http.NewRequest("GET", "http://localhost:8080", nil)
	require.NoError(t, err)

	resp, err := b.RoundTrip(r)
	require.NoError(t, err)
	require.Equal(t, resp.Request.URL.Host, host)
}

func TestBackendShouldWrapErrorWithBackendError(t *testing.T) {
	host := "someremote.backend:8080"
	netURL, err := url.Parse(fmt.Sprintf("http://%s", host))
	require.NoError(t, err)

	hostURL := types.YAMLUrl{URL: netURL}
	roundtripper := func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("Connection timeout")
	}

	backendConfig := config.Backend{Endpoint: hostURL, Type: "passthrough"}
	b, err := newBackend(backendConfig, prepareTestTransportContainer(roundtripper))
	require.NoError(t, err)

	r, err := http.NewRequest("GET", "http://localhost:8080", nil)
	require.NoError(t, err)

	resp, err := b.RoundTrip(r)
	require.Error(t, err)
	require.Nil(t, resp)

	berr, ok := err.(utils.BackendError)
	require.True(t, ok)
	require.Equal(t, host, berr.Backend())
}

func prepareTestTransportContainer(roundtripper func(*http.Request) (*http.Response, error)) transport.Container {
	return transport.Container{
		RoundTrippers: map[string]http.RoundTripper{
			"DefaultTransport": &testRt{rt: roundtripper},
		},
		TransportsConfig: transportConfig.Transports{transportConfig.Transport{
			Name: "DefaultTransport",
			Matchers: transportConfig.ClientTransportMatchers{
				Method:     "",
				Path:       "",
				QueryParam: "",
			},
		},
		},
	}
}
