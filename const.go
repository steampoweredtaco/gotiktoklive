package gotiktoklive

import (
	"errors"
	"regexp"
)

const (
	// Base URL
	tiktokBaseUrl = "https://www.tiktok.com/"
	tiktokAPIUrl  = "https://webcast.tiktok.com/webcast/"

	// Endpoints
	urlLive      = "live/"
	urlFeed      = "feed/"
	urlRankList  = "ranklist/online_audience/"
	urlPriceList = "wallet_api/fs/diamond"
	urlUser      = "@%s/"
	// Think this changed to room/enter/
	urlRoomInfo = "room/info/"
	urlRoomData = "webcast/fetch/"
	urlGiftInfo = "gift/list/"
	// added slash is intention to simplify the base signer url which is now configurable.
	urlSignReq = "/webcast/fetch/"

	urlCheckLive      = "room/check_alive/"
	clientNameDefault = "gotiktok_live"
	apiKeyDefault     = ""
	userAgent         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.5005.63 Safari/537.36"
	referer           = "https://www.tiktok.com/"
	// webcast backend will fail if origin is an ending /
	origin = "https://www.tiktok.com"
)

var (
	// Used for webcast messages that don't require anything fancy
	minGetParams = map[string]string{
		"aid":          "1988",
		"app_language": "en-US",
		"app_name":     "tiktok_web",
	}
	defaultGETParams = map[string]string{
		"aid":                 "1988",
		"app_language":        "en-US",
		"app_name":            "tiktok_web",
		"browser_language":    "en",
		"browser_name":        "Mozilla",
		"browser_online":      "true",
		"browser_platform":    "Win32",
		"browser_version":     userAgent,
		"cookie_enabled":      "true",
		"cursor":              "",
		"internal_ext":        "",
		"device_platform":     "web",
		"focus_state":         "true",
		"from_page":           "user",
		"history_len":         "4",
		"is_fullscreen":       "false",
		"is_page_visible":     "true",
		"did_rule":            "3",
		"fetch_rule":          "1",
		"last_rtt":            "0",
		"live_id":             "12",
		"resp_content_type":   "protobuf",
		"screen_height":       "1152",
		"screen_width":        "2048",
		"tz_name":             "Europe/Berlin",
		"referer":             referer,
		"root_referer":        origin,
		"msToken":             "",
		"version_code":        "180800",
		"webcast_sdk_version": "1.3.0",
		"update_version_code": "1.3.0",
	}
	defaultRequestHeeaders = map[string]string{
		"Connection":    "keep-alive",
		"Cache-control": "max-age=0",
		"User-Agent":    userAgent,
		"Accept":        "text/html,application/json,application/protobuf",
		"Referer":       referer,
		"Origin":        origin,
		// clientId:  = "ttlive-golang"
		"Accept-Language": "en-US,en;q=0.9",
		"Accept-Encoding": "gzip, deflate",
	}
	reJsonData = []*regexp.Regexp{
		regexp.MustCompile(`<script id="SIGI_STATE"[^>]+>(.*?)</script>`),
		regexp.MustCompile(`<script id="sigi-persisted-data">window\['SIGI_STATE'\]=(.*);w`),
	}
	reVerify = regexp.MustCompile(`tiktok-verify-page`)
)

var (
	ErrUserOffline       = errors.New("user might be offline, Room ID not found")
	ErrIPBlocked         = errors.New("your IP or country might be blocked by TikTok")
	ErrLiveHasEnded      = errors.New("livestream has ended")
	ErrMsgNotImplemented = errors.New("message protobuf type has not been implemented, please report")
	ErrNoMoreFeedItems   = errors.New("no more feed items available")
	ErrUserNotFound      = errors.New("user not found")
	ErrCaptcha           = errors.New("captcha detected, unable to proceed")
	ErrURLNotFound       = errors.New("unable to download stream, URL not found")
	ErrFFMPEGNotFound    = errors.New("please install ffmpeg before downloading")
	ErrRateLimitExceeded = errors.New("you have exceeded the rate limit, please wait a few min")
	ErrUserInfoNotFound  = errors.New("user info not found")
)
