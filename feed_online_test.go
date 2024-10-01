//go:build requiresOnline

package gotiktoklive

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"

	"github.com/steampoweredtaco/gotiktoklive/test_types"
)

func TestFeedItem(t *testing.T) {
	tiktok, err := NewTikTok()
	if !assert.NoError(t, err) {
		return
	}
	feed := tiktok.NewFeed()

	items := []*LiveStream{}
	i := 0
	for {
		feedItem, err := feed.Next()
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, feedItem.LiveStreams...)
		i++
		t.Logf("%d : %d, %v", feedItem.Extra.MaxTime, len(feedItem.LiveStreams), feedItem.Extra.HasMore)
		for _, stream := range feedItem.LiveStreams {
			t.Logf("%s : %d viewers, %s", stream.Room.Owner.Nickname, stream.Room.UserCount, stream.LiveReason)
		}

		if !feedItem.Extra.HasMore || i > 5 {
			break
		}
	}
	t.Logf("Found %d items, over %d requests", len(items), i)

	sort.Slice(items, func(i, j int) bool {
		return items[i].Room.UserCount > items[j].Room.UserCount
	})

	test_types.USERNAME = items[0].Room.Owner.Username
	t.Logf("Setting username to %s", items[0].Room.Owner.Username)
}
