//go:build requiresOnline

package gotiktoklive

import (
	"log/slog"
	"testing"
	"time"

	"github.com/steampoweredtaco/gotiktoklive/test_types"
)

func TestLiveTrackUser(t *testing.T) {
	tiktok, err := NewTikTok()
	if err != nil {
		t.Fatal(err)
	}
	live, err := tiktok.TrackUser(test_types.USERNAME)
	if err != nil {
		t.Fatal(err)
	}

	timeout := time.After(5 * time.Second)

	eventCounter := 0
	for {
		select {
		case event := <-live.Events:
			t.Logf("%T", event)
			eventCounter++
		case <-timeout:
			if eventCounter < 10 {
				t.Fatal("Less than 10 events received.. Something seems off")
			}
			return
		}
	}
}

func TestLivePriceList(t *testing.T) {
	tiktok, err := NewTikTok()
	if err != nil {
		t.Fatal(err)
	}
	priceList, err := tiktok.GetPriceList()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Fetched %d prices", len(priceList.PriceList))
}

func TestLiveDownload(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	tiktok, err := NewTikTok(SigningApiKey(test_types.APIKEY))
	if err != nil {
		t.Fatal(err)
	}
	live, err := tiktok.TrackUser(test_types.APIKEY)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer func() {
			done <- struct{}{}
		}()
		err = live.DownloadStream()
		if err != nil {
			t.Fatal(err)
		}
	}()

	select {
	case _ = <-done:
		t.Log("command exited")
	case <-time.After(12 * time.Second):
		t.Log("test complete")
	}
	live.Close()

	live, err = tiktok.TrackUser(test_types.APIKEY)
	if err != nil {
		t.Fatal(err)
	}

	done = make(chan struct{})
	go func() {
		defer func() {
			done <- struct{}{}
		}()
		err = live.DownloadStream("my-test-download.mkv")
		if err != nil {
			t.Fatal(err)
		}
	}()
	select {
	case _ = <-done:
		t.Log("command exited")
	case <-time.After(4 * time.Hour):
		t.Log("test complete")
	}
	live.Close()
}

// func TestRankList(t *testing.T) {
// 	tiktok := gotiktoklive.NewTikTok()
// 	live, err := tiktok.TrackUser(USERNAME)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	tiktok.LogRequests = true
// 	rankList, err := live.GetRankList()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	if len(rankList.Ranks) == 0 {
// 		t.Fatal("No ranked users found")
// 	}
//
// 	topUser := rankList.Ranks[0]
// 	t.Logf("Top user (%s) has donated %d coins", topUser.User.Nickname, topUser.Score)
// }
