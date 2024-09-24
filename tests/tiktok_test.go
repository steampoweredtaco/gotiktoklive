package tests

import (
	"testing"

	"github.com/steampoweredtaco/gotiktoklive"
)

func TestUserInfo(t *testing.T) {
	tiktok, err := gotiktoklive.NewTikTok()
	if err != nil {
		t.Fatal(err)
	}

	info, err := tiktok.GetUserInfo(USERNAME)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Got user info: %+v", info)
}

func TestRoomInfo(t *testing.T) {
	tiktok, err := gotiktoklive.NewTikTok()
	if err != nil {
		t.Fatal(err)
	}
	info, err := tiktok.GetRoomInfo(USERNAME)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Got user info: %+v", info)
}
