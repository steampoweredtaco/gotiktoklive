//go:build requiresOnline

package gotiktoklive

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"

	"github.com/steampoweredtaco/gotiktoklive/test_types"
	"golang.org/x/net/context"
)

func TestWebsocket(t *testing.T) {
	tiktok, err := NewTikTok()
	if !assert.NoError(t, err) {
		return
	}
	tiktok.Debug = true
	tiktok.debugHandler = func(i ...interface{}) {
		t.Log(i...)
	}
	id, err := tiktok.getRoomID(test_types.USERNAME)
	if !assert.NoError(t, err) {
		return
	}

	live := Live{
		t:      tiktok,
		ID:     id,
		wg:     &sync.WaitGroup{},
		Events: make(chan interface{}, 100),
	}

	ctx, cancel := context.WithCancel(context.Background())
	live.done = ctx.Done
	live.close = func() {
		cancel()
		close(live.Events)
	}

	err = live.getRoomData()
	if !assert.NoError(t, err) {
		return
	}

	if live.wsURL == "" {
		t.Fatal("No websocket url provided")
	}
	t.Logf("Ws url: %s, %+v", live.wsURL, live.wsParams)

	if err := live.connect(live.wsURL, live.wsParams); err != nil {
		t.Fatal(err)
	}

	tiktok.wg.Add(2)
	go live.readSocket()
	go live.sendPing()

	timeout := time.After(5 * time.Second)
	for {
		select {
		case <-timeout:
			return
		case event := <-live.Events:
			switch e := event.(type) {
			case UserEvent:
				t.Logf("%T: %s (%s) %s", e, e.User.Nickname, e.User.Nickname, e.Event)
			default:
				t.Logf("%T: %+v", e, e)
			}
		}
	}
}
