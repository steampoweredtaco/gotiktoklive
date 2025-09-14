package gotiktoklive

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"
)

type loggingTransport struct {
	Transport http.RoundTripper
}

func (s *loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	bytes, _ := httputil.DumpRequestOut(r, true)

	resp, err := s.Transport.RoundTrip(r)
	// err is returned after dumping the response
	if err != nil && strings.Contains(err.Error(), "malformed HTTP response") {
		httputil.DumpResponse(resp, false)
		slog.Debug(fmt.Sprintf("%s\n", bytes))
		return resp, err
	}
	if resp == nil {
		return resp, err
	}
	respBytes, _ := httputil.DumpResponse(resp, true)
	bytes = append(bytes, respBytes...)

	slog.Debug(fmt.Sprintf("%s\n", bytes))

	return resp, err
}
