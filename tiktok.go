package gotiktoklive

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/ratelimit"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/cookiejar"
	neturl "net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	defaultSignerURL = "https://tiktok.eulerstream.com"
)

// TikTok allows you to track and discover current live streams.
type TikTok struct {
	c    *http.Client
	wg   *sync.WaitGroup
	done func() <-chan struct{}

	streams int
	mu      *sync.Mutex

	// Pass extra debug messages to debugHandler
	Debug bool

	// LogRequests when set to true will log all made requests in JSON to debugHandler
	LogRequests bool

	infoHandler  func(...interface{})
	warnHandler  func(...interface{})
	debugHandler func(...interface{})
	errHandler   func(...interface{})

	proxy                    *neturl.URL
	apiKey                   string
	clientName               string
	shouldReconnect          bool
	enableExperimentalEvents bool
	enableExtraDebug         bool
	enableWSTrace            bool
	wsTraceFile              string
	wsTraceChan              chan struct{ direction, hex string }
	wsTraceOut               *bufio.Writer
	signerUrl                string
	getLimits                bool
	limiter                  ratelimit.Limiter
}

// NewTikTok creates a tiktok instance that allows you to track live streams and
//
//	discover current livestreams.
func NewTikTok(options ...TikTokLiveOption) (*TikTok, error) {
	return NewTikTokWithApiKey(clientNameDefault, apiKeyDefault, options...)
}

// NewTikTokWithApiKey allows to use an ApiKey with the default signer.
func NewTikTokWithApiKey(clientName, apiKey string, options ...TikTokLiveOption) (*TikTok, error) {
	jar, _ := cookiejar.New(nil)
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	tiktok := TikTok{
		c: &http.Client{
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			// Transport: &loggingTransport{},
		},
		wg:              &wg,
		done:            ctx.Done,
		mu:              &sync.Mutex{},
		infoHandler:     defaultLogHandler,
		warnHandler:     defaultLogHandler,
		debugHandler:    routineErrHandler,
		errHandler:      routineErrHandler,
		signerUrl:       defaultSignerURL,
		clientName:      clientName,
		apiKey:          apiKey,
		shouldReconnect: true,
		getLimits:       true,
	}
	envs := []string{"HTTP_PROXY", "HTTPS_PROXY"}
	var optionsErr []error

	for _, env := range envs {
		if e := os.Getenv(env); e != "" {
			tiktok.setProxy(e, false)
		}
	}
	for _, option := range options {
		optionsErr = append(optionsErr, option(&tiktok))
	}
	err := errors.Join(optionsErr...)
	if err != nil {
		cancel()
		return nil, err
	}
	if tiktok.getLimits {
		limits, err := GetSignerLimits(tiktok.signerUrl, tiktok.apiKey)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("cannot get signing limits: %w", err)
		}
		slog.Debug("limits found, using per minute limit", "day", limits.Day, "hour", limits.Hour, "minute", limits.Minute)
		limiter := ratelimit.New(limits.Minute.Max, ratelimit.Per(1*time.Minute), ratelimit.WithoutSlack)
		tiktok.limiter = limiter
	} else {
		slog.Debug("Request limits set to sane default of 10 per minute, for more enable GetLimits option to use signer specified limits")
		limiter := ratelimit.New(10, ratelimit.Per(1*time.Minute), ratelimit.WithoutSlack)
		tiktok.limiter = limiter
	}

	if tiktok.enableWSTrace {
		var err error
		tiktok.wsTraceFile, err = filepath.Abs(tiktok.wsTraceFile)
		if err != nil {
			tiktok.errHandler(fmt.Errorf("cannot get info for ws trace file, it will not be enable: %w", err))
			tiktok.enableWSTrace = false
			goto continueSetup
		}
		f, err := os.Create(tiktok.wsTraceFile)
		tiktok.wsTraceOut = bufio.NewWriter(f)

		wg.Add(1)
		go func() {
			defer func() {
				_ = f.Close()
			}()
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case t := <-tiktok.wsTraceChan:
					timestamp := time.Now().UTC().Format("2006-01-02 15:04:05.000")
					tiktok.wsTraceOut.Write([]byte(timestamp))
					tiktok.wsTraceOut.Write([]byte(t.direction))
					tiktok.wsTraceOut.Write([]byte(" "))
					tiktok.wsTraceOut.Write([]byte(t.hex))
					tiktok.wsTraceOut.Write([]byte("\n"))
					tiktok.wsTraceOut.Flush()
				}
			}
		}()
	}
continueSetup:
	setupInterruptHandler(
		func(c chan os.Signal) {
			<-c
			cancel()
			wg.Wait()

			tiktok.infoHandler("Shutting down...")
			os.Exit(0)
		})

	tiktok.sendRequest(&reqOptions{
		OmitAPI: true,
	}, nil)

	return &tiktok, nil
}

// GetLiveRoomUserInfo will fetch information about the user's live room which contains
// information about the user and also the live room which contains their user ID, as well
// as the RoomID, with which you can tell if they are live.
func (t *TikTok) GetLiveRoomUserInfo(user string) (LiveRoomUserInfo, error) {
	user = cleanupUser(user)
	body, _, err := t.sendRequest(&reqOptions{
		Endpoint: fmt.Sprintf(urlUser+urlLive, user),
		Query:    defaultRequestHeeaders,
		OmitAPI:  true,
	}, func(response *http.Response) error {
		if response.StatusCode != 200 {
			return UserNotFound{}
		}
		return nil
	})
	if err != nil {
		return LiveRoomUserInfo{}, err
	}

	// Find json data in HTML page
	var matches [][]byte
	for _, re := range reJsonData {
		matches = re.FindSubmatch(body)
		if len(matches) != 0 {
			break
		}
	}
	if len(matches) == 0 {
		return LiveRoomUserInfo{}, &ErrIPBlockedOrBanned{}
	}

	// Parse json data
	var res struct {
		LiveRoom *liveRoomContainer `json:"liveRoom,omitempty"`
	}
	if err := json.Unmarshal(matches[1], &res); err != nil {
		return LiveRoomUserInfo{}, err
	}

	if res.LiveRoom == nil || res.LiveRoom.LiveRoomUserInfo == nil {
		return LiveRoomUserInfo{}, ErrUserNotFound
	}

	return *res.LiveRoom.LiveRoomUserInfo, nil
}

func cleanupUser(user string) string {
	return strings.TrimLeft(user, "@")
}

// GetUserInfo will fetch information about the user, such as followers stats,
//
//	their user ID, as well as the RoomID, with which you can tell if they are live.
func (t *TikTok) GetUserInfo(user string) (LiveRoomUser, error) {
	roomUserInfo, err := t.GetLiveRoomUserInfo(user)
	if err != nil {
		return LiveRoomUser{}, err
	}
	return *roomUserInfo.LiveRoomUser, nil
}

// GetPriceList fetches the price list of tiktok coins. Prices will be given in
//
//	USD cents and the cents equivalent of the local currency of the IP location.
//
// To fetch a different currency, use a VPN or proxy to change your IP to a
//
//	different country.
func (t *TikTok) GetPriceList() (*PriceList, error) {
	body, _, err := t.sendRequest(&reqOptions{
		Endpoint: urlPriceList,
		Query:    defaultGETParams,
	}, nil)
	if err != nil {
		return nil, err
	}

	var rsp PriceList
	if err := json.Unmarshal(body, &rsp); err != nil {
		return nil, err
	}

	return &rsp, nil
}

func (t *TikTok) SetInfoHandler(f func(...interface{})) {
	t.infoHandler = f
}

func (t *TikTok) SetWarnHandler(f func(...interface{})) {
	t.warnHandler = f
}

func (t *TikTok) SetDebugHandler(f func(...interface{})) {
	t.debugHandler = f
}

func (t *TikTok) SetErrorHandler(f func(...interface{})) {
	t.errHandler = f
}

// setProxy will set a proxy for both the http client as well as the websocket.
// You can manually set a proxy with this method, or by using the HTTPS_PROXY
//
//	environment variable.
//
// ALL_PROXY can be used to set a proxy only for the websocket.
func (t *TikTok) setProxy(url string, insecure bool) error {
	uri, err := neturl.Parse(url)
	if err != nil {
		return err
	}

	t.proxy = uri

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: insecure,
	}
	tr.Proxy = http.ProxyURL(uri)
	if originalClient, ok := t.c.Transport.(*loggingTransport); ok {
		originalClient.transport = tr
	} else {
		t.c.Transport = tr
	}

	// t.c.Transport = &http.Transport{
	// 	Proxy: http.ProxyURL(uri),
	// 	TLSClientConfig: &tls.Config{
	// 		InsecureSkipVerify: insecure,
	// 	},
	// }
	return nil
}

// IsLive can determine if the user is live. Use GetLiveRoomUserInfo first, if
// user is not found that means there was never a live by that user in the first
// place.
func (t *TikTok) IsLive(info LiveRoomUserInfo) (bool, error) {
	minGetParams := maps.Clone(minGetParams)
	minGetParams["room_ids"] = info.LiveRoomUser.RoomID

	type DataItem struct {
		Alive     bool   `json:"alive"`
		RoomID    int64  `json:"room_id"`
		RoomIDStr string `json:"room_id_str"`
	}

	type Extra struct {
		Now int64 `json:"now"`
	}

	type Response struct {
		Data       []DataItem `json:"data"`
		Extra      Extra      `json:"extra"`
		StatusCode int        `json:"status_code"`
	}

	body, _, err := t.sendRequest(&reqOptions{
		Endpoint: urlCheckLive,
		Query:    minGetParams,
		OmitAPI:  false,
	}, nil)
	if err != nil {
		return false, err
	}
	var res Response
	if err := json.Unmarshal(body, &res); err != nil {
		return false, err
	}

	for i := range res.Data {
		if res.Data[i].RoomIDStr == info.LiveRoomUser.RoomID {
			return res.Data[i].Alive, nil
		}
	}
	return false, fmt.Errorf("roomID not found in result")
}

func setupInterruptHandler(f func(chan os.Signal)) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go f(c)
}

// GetSignerLimits returns the limits as provided by the signer.  This supports eulerstream signer and any other signers that
// support the same API endpoints
//
// Example:
//
//	limits, _ := GetSignerLimits("https://tiktok.eulerstream.com", "MyApiKey")
//	fmt.Printf("limits Day: %d, Hour: %d, Minutes %d\n", limits.Day, limits.Hour, limits.Minute)
func GetSignerLimits(signer string, apiKey string) (SigningLimits, error) {
	resp, err := http.Get(fmt.Sprintf("%s/webcast/rate_limits?apiKey=%s", signer, apiKey))
	if err != nil {
		return SigningLimits{}, fmt.Errorf("cannot get rate_limts: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return SigningLimits{}, fmt.Errorf("bad status getting rate_limts %s(%d), cannot read body: %w", resp.Status, resp.StatusCode, err)
		}
		return SigningLimits{}, fmt.Errorf("bad status getting rate_limits %s(%d), %s", resp.Status, resp.StatusCode, string(body))
	}
	jsonR := resp.Body
	defer func(jsonR io.ReadCloser) {
		_ = jsonR.Close()
	}(jsonR)
	limits := SigningLimits{}
	err = json.NewDecoder(jsonR).Decode(&limits)
	if err != nil {
		return SigningLimits{}, err
	}
	return limits, nil
}
