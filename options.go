package gotiktoklive

type TikTokLiveOption func(t *TikTok)

// SigningApiKey sets the singer API key.
func SigningApiKey(apiKey string) TikTokLiveOption {
	return func(t *TikTok) {
		t.apiKey = apiKey
	}
}

// EnableExperimentalEvents enables experimental events that have not been figured out yet and the API for them
// is not stable.  It may also induce additional logging that might be undesirable.
func EnableExperimentalEvents(t *TikTok) {
	t.enableExperimentalEvents = true
}
