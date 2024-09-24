package gotiktoklive

type TikTokLiveOption func(t *TikTok)

// SigningApiKey sets the singer API key.
func SigningApiKey(apiKey string) TikTokLiveOption {
	return func(t *TikTok) {
		t.apiKey = apiKey
	}
}

// SigningUrl defines the signer. The default is https://tiktok.eulerstream.com. Supports any signer that supports the
// signing api as defined by https://www.eulerstream.com/docs/openapi
func SigningUrl(url string) TikTokLiveOption {
	return func(t *TikTok) {
		t.signerUrl = url
	}
}

// DisableSigningLimitsValidation will disable querying the signer for limits and using those as the reasonable limits
// for signing requests per second. Instead, this library will be limited to signing only 5 signing requests per minute
// and may limit functionality compared to the request limit the signer provides.
func DisableSigningLimitsValidation(t *TikTok) {
	t.getLimits = false
}

// EnableExperimentalEvents enables experimental events that have not been figured out yet and the API for them is not
// stable. It may also induce additional logging that might be undesirable.
func EnableExperimentalEvents(t *TikTok) {
	t.enableExperimentalEvents = true
}

// EnableExtraWebCastDebug an unreasonable amount of debug for library development and troubleshooting. This option
// makes no guarantee of ever having the same output and is only for development and triage purposes.
func EnableExtraWebCastDebug(t *TikTok) {
	t.enableExtraDebug = true
}

// EnableWSTrace will put traces for all websocket messages into the given file. The file will be overwritten so
// if you want multiple traces make sure handle giving a unique filename each startup.
func EnableWSTrace(file string) func(t *TikTok) {
	return func(t *TikTok) {
		t.enableWSTrace = true
		t.wsTraceFile = file
		t.wsTraceChan = make(chan struct{ direction, hex string }, 50)
	}
}
