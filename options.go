package gotiktoklive

type TikTokLiveOption func(t *TikTok)

// DoNotAutoReconnect prevents auto retrying to reconnect to the TikTok webcast backend once after a failure, which is
// the default behavior.  This is useful for if you have a program trying to monitor and manage the reconnections by
// monitoring the live Events channel for closure.
func DoNotAutoReconnect(t *TikTok) {
	t.shouldReconnect = false
}

// SigningApiKey sets the singer API key.
func SigningApiKey(apiKey string) TikTokLiveOption {
	return func(t *TikTok) {
		t.apiKey = apiKey
	}
}
