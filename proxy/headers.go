package proxy

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/viper"
	"net/http"
	"net/textproto"
	"strings"
)

// isReserved returns whether the key is reserved by gRPC.
func isReserved(key string) bool {
	return strings.HasPrefix(key, "Grpc-")
}

// incomingHeaderMatcher converts an HTTP header name on http.Request to
// grpc metadata. Permanent headers (i.e. User-Agent) are prepended with
// "grpc-gateway". Headers that start with start with "Grpc-" (reserved
// by grpc) are prepended with "X-". Other headers are forwarded as is.
func incomingHeaderMatcher(key string) (string, bool) {
	key = textproto.CanonicalMIMEHeaderKey(key)
	if isPermanentHTTPHeader(key) {
		return runtime.MetadataPrefix + key, true
	}
	if isReserved(key) {
		return "X-" + key, true
	}

	// The Istio service mesh dislikes when you pass the Content-Length header
	if key == "Content-Length" {
		return "", false
	}

	return key, true
}

// outgoingHeaderMatcher transforms outgoing metadata into HTTP headers.
// We return any response metadata as is.
func outgoingHeaderMatcher(metadata string) (string, bool) {
	return metadata, true
}

func allowCors(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		corsAllowOrigin := viper.GetString("proxy.allow_origin")
		if corsAllowOrigin == "*" {
			if origin := req.Header.Get("Origin"); origin != "" {
				corsAllowOrigin = origin
			}
		}
		w.Header().Set("Access-Control-Allow-Origin", corsAllowOrigin)
		w.Header().Set("Access-Control-Allow-Credentials", viper.GetString("proxy.allow_creds"))
		w.Header().Set("Access-Control-Allow-Methods", viper.GetString("proxy.allow_methods"))
		w.Header().Set("Access-Control-Allow-Headers", viper.GetString("proxy.allow_headers"))
		if req.Method == "OPTIONS" && req.Header.Get("Access-Control-Request-Method") != "" {
			return
		}
		handler.ServeHTTP(w, req)
	})
}

// sanitizeApiPrefix forces prefix to be non-empty and end with a slash.
func sanitizeApiPrefix(prefix string) string {
	if len(prefix) == 0 || prefix[len(prefix)-1:] != "/" {
		return prefix + "/"
	}
	return prefix
}

// isPermanentHTTPHeader checks whether hdr belongs to the list of
// permenant request headers maintained by IANA.
// http://www.iana.org/assignments/message-headers/message-headers.xml
// From https://github.com/grpc-ecosystem/grpc-gateway/blob/7a2a43655ccd9a488d423ea41a3fc723af103eda/runtime/context.go#L157
func isPermanentHTTPHeader(hdr string) bool {
	switch hdr {
	case
		"Accept",
		"Accept-Charset",
		"Accept-Language",
		"Accept-Ranges",
		"Authorization",
		"Cache-Control",
		"Content-Type",
		"Cookie",
		"Date",
		"Expect",
		"From",
		"Host",
		"If-Match",
		"If-Modified-Since",
		"If-None-Match",
		"If-Schedule-Tag-Match",
		"If-Unmodified-Since",
		"Max-Forwards",
		"Origin",
		"Pragma",
		"Referer",
		"User-Agent",
		"Via",
		"Warning":
		return true
	}
	return false
}
