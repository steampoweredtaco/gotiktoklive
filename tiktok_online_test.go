//go:build requiresOnline

package gotiktoklive

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/steampoweredtaco/gotiktoklive/test_types"
)

func TestRoomID(t *testing.T) {
	type test struct {
		username string
	}
	tests := map[string]test{
		"basic test": {
			username: test_types.USERNAME,
		},
		"@ test": {
			username: "@" + test_types.USERNAME,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(tt *testing.T) {
			tiktok, err := NewTikTok(SetProxy(test_types.PROXY, test_types.PROXY_INSECURE))
			if !assert.NoError(tt, err) {
				return
			}
			id, err := tiktok.getRoomID(test.username)
			if !assert.NoError(tt, err) {
				return
			}
			t.Logf("Found Room ID: %s ", id)
		})
	}
}

func TestRoomInfo(t *testing.T) {
	type test struct {
		username string
	}
	tests := map[string]test{
		"basic test": {
			username: test_types.USERNAME,
		},
		"@ test": {
			username: "@" + test_types.USERNAME,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(tt *testing.T) {
			tiktok, err := NewTikTok(SetProxy(test_types.PROXY, test_types.PROXY_INSECURE))
			if !assert.NoError(tt, err) {
				return
			}
			id, err := tiktok.getRoomID(test.username)
			if !assert.NoError(tt, err) {
				return
			}
			live := Live{
				t:  tiktok,
				ID: id,
			}
			info, err := live.getRoomInfo()
			if !assert.NoError(tt, err) {
				return
			}
			tt.Log(info.Title)
		})
	}
}

func TestGiftInfo(t *testing.T) {
	type test struct {
		username string
	}
	tests := map[string]test{
		"basic test": {
			username: test_types.USERNAME,
		},
		"@ test": {
			username: "@" + test_types.USERNAME,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(tt *testing.T) {
			tiktok, err := NewTikTok(SetProxy(test_types.PROXY, test_types.PROXY_INSECURE))
			if !assert.NoError(tt, err) {
				return
			}
			id, err := tiktok.getRoomID(test.username)
			if !assert.NoError(tt, err) {
				return
			}

			live := Live{
				t:  tiktok,
				ID: id,
			}

			info, err := live.getGiftInfo()
			if !assert.NoError(tt, err) {
				return
			}
			t.Logf("Found %d gifts", len(info.Gifts))
		})
	}
}

func TestRoomData(t *testing.T) {
	type test struct {
		username string
	}
	tests := map[string]test{
		"basic test": {
			username: test_types.USERNAME,
		},
		"@ test": {
			username: "@" + test_types.USERNAME,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(tt *testing.T) {
			tiktok, err := NewTikTok(SetProxy(test_types.PROXY, test_types.PROXY_INSECURE))
			if !assert.NoError(tt, err) {
				return
			}
			id, err := tiktok.getRoomID(test.username)
			if !assert.NoError(tt, err) {
				return
			}
			tiktok.streams++
			live := Live{
				t:      tiktok,
				ID:     id,
				Events: make(chan interface{}, 100),
			}

			err = live.getRoomData()
			if !assert.NoError(tt, err) {
				return
			}
			t.Logf("Ws url: %s, %+v", live.wsURL, live.wsParams)
		})
	}
}

func TestUserInfo(t *testing.T) {
	type test struct {
		username string
	}
	tests := map[string]test{
		"basic test": {
			username: test_types.USERNAME,
		},
		"@ test": {
			username: "@" + test_types.USERNAME,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(tt *testing.T) {
			tiktok, err := NewTikTok(SetProxy(test_types.PROXY, test_types.PROXY_INSECURE))
			if !assert.NoError(tt, err) {
				return
			}

			info, err := tiktok.GetUserInfo(test.username)
			if !assert.NoError(tt, err) {
				return
			}

			t.Logf("Got user info: %+v", info)
		})
	}
}

func TestLiveRoomUser(t *testing.T) {
	type test struct {
		username string
	}
	tests := map[string]test{
		"basic test": {
			username: test_types.USERNAME,
		},
		"@ test": {
			username: "@" + test_types.USERNAME,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(tt *testing.T) {
			tiktok, err := NewTikTok(SetProxy(test_types.PROXY, test_types.PROXY_INSECURE))

			if !assert.NoError(tt, err) {
				return
			}

			info, err := tiktok.GetLiveRoomUserInfo(test.username)
			if !assert.NoError(tt, err) {
				return
			}

			t.Logf("Got user info: %+v", info)
		})
	}
}

func TestIsLive(t *testing.T) {
	tiktok, err := NewTikTok(SetProxy(test_types.PROXY, test_types.PROXY_INSECURE))
	if !assert.NoError(t, err) {
		return
	}

	info, err := tiktok.GetLiveRoomUserInfo(test_types.USERNAME)
	if !assert.NoError(t, err) {
		return
	}
	t.Logf("Got user info: %+v", info)
	isLive, err := tiktok.IsLive(info)
	if !assert.NoError(t, err) {
		return
	}
	t.Logf("User is live: %+v", isLive)
}
