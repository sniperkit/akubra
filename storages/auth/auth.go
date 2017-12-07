package auth

import (
	"fmt"
	"net/http"

	"github.com/allegro/akubra/httphandler"
)

const (
	// Passthrough is basic type, does nothing to the request
	Passthrough = "passthrough"
	// S3FixedKey will sign requests with single key
	S3FixedKey = "S3FixedKey"
	// S3AuthService will sign requests using key from external source
	S3AuthService = "S3AuthService"
)

// Decorators maps Backend type with httphadler decorators factory
var Decorators = map[string]func(map[string]string, string) (httphandler.Decorator, error){
	Passthrough: func(map[string]string, string) (httphandler.Decorator, error) {
		return func(rt http.RoundTripper) http.RoundTripper {
			return rt
		}, nil
	},
	S3FixedKey: func(properties map[string]string, backend string) (httphandler.Decorator, error) {
		accessKey, ok := properties["AccessKey"]
		if !ok {
			return nil, fmt.Errorf("no AccessKey defined for backend type %q", S3FixedKey)
		}

		secret, ok := properties["Secret"]
		if !ok {
			return nil, fmt.Errorf("no Secret defined for backend type %q", S3FixedKey)
		}

		k := Keys{
			AccessKeyID:     accessKey,
			SecretAccessKey: secret,
		}
		return SignDecorator(k), nil
	},
	S3AuthService: func(properties map[string]string, backend string) (httphandler.Decorator, error) {
		endpoint, ok := properties["AuthServiceEndpoint"]
		if !ok {
			endpoint = "default"
		}

		return SignAuthServiceDecorator(backend, endpoint), nil
	},
}