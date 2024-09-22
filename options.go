package gotiktoklive

type TikTokLiveOption func(t *TikTok)

// SigningApiKey sets the singer API key.
func SigningApiKey(apiKey string) TikTokLiveOption {
	return func(t *TikTok) {
		t.apiKey = apiKey
	}
}

// EnableExperimentalEvents enables experimental events that have not been
// figured out yet and the API for them is not stable. It may also induce
// additional logging that might be undesirable.
func EnableExperimentalEvents(t *TikTok) {
	t.enableExperimentalEvents = true
}

// EnableExtraWebCastDebug an unreasonable amount of debug for library
// development and troubleshooting. This option makes no guarantee of ever having
// the same output and is only for development and triage purposes.
func EnableExtraWebCastDebug(t *TikTok) {
	t.enableExtraDebug = true
}

// EnableWSTrace will put traces for all websocket messages into the give file. The file will be overwritten so
// if you want multiple traces make sure handle giving a unique filename each startup.
func EnableWSTrace(file string) func(t *TikTok) {
	return func(t *TikTok) {
		t.enableWSTrace = true
		t.wsTraceFile = file
		t.wsTraceChan = make(chan struct{ direction, hex string }, 50)
	}
}
