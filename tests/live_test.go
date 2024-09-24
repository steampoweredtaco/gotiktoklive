package tests

import (
	"log/slog"
	"testing"
	"time"

	"github.com/steampoweredtaco/gotiktoklive"
)

func TestLiveTrackUser(t *testing.T) {
	tiktok, err := gotiktoklive.NewTikTok()
	if err != nil {
		t.Fatal(err)
	}
	live, err := tiktok.TrackUser(USERNAME)
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
	tiktok, err := gotiktoklive.NewTikTok()
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
	tiktok, err := gotiktoklive.NewTikTok()
	if err != nil {
		t.Fatal(err)
	}
	live, err := tiktok.TrackUser(USERNAME)
	if err != nil {
		t.Fatal(err)
	}
	err = live.DownloadStream()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Second)
	live.Close()

	live, err = tiktok.TrackUser(USERNAME)
	if err != nil {
		t.Fatal(err)
	}
	slog.SetLogLoggerLevel(slog.LevelDebug)
	err = live.DownloadStream("my-test-download.mkv")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Minute)
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
