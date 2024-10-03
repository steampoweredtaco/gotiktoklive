package gotiktoklive

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"

	pb "github.com/steampoweredtaco/gotiktoklive/proto"

	"google.golang.org/protobuf/proto"
)

// TODO: check gift prices of gifts not in wish list

const (
	DEFAULT_EVENTS_CHAN_SIZE = 100
)

// Live allows you to track a livestream.
// To track a user call tiktok.TrackUser(<user>).
type Live struct {
	t *TikTok

	cursor   string
	wss      net.Conn
	wsURL    string
	wsParams map[string]string
	close    func()
	done     func() <-chan struct{}
	cancel   context.CancelFunc

	ID       string
	Info     *RoomInfo
	GiftInfo *GiftInfo
	Events   chan Event
	chanSize int
	wg       *sync.WaitGroup
}

func (t *TikTok) newLive(roomId string) *Live {
	live := Live{
		t:        t,
		ID:       roomId,
		wg:       &sync.WaitGroup{},
		Events:   make(chan Event, DEFAULT_EVENTS_CHAN_SIZE),
		chanSize: DEFAULT_EVENTS_CHAN_SIZE,
	}
	t.mu.Lock()
	t.streams += 1
	t.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	live.cancel = cancel
	live.done = ctx.Done
	o := sync.Once{}
	live.close = func() {
		o.Do(func() {
			// cancel needs to be first as live.done is used to know to exit in all the
			// various goroutines which should release the waitgroup. It is ok for anywhere
			// to call cancel to trigger the other routines, but calls to close is only for
			// cleanup and block till done
			cancel()
			live.wss.Close()
			live.wg.Wait()
			t.mu.Lock()
			t.streams -= 1
			t.mu.Unlock()
		})
	}

	return &live
}

// Close will terminate the connection and stop any downloads.
func (l *Live) Close() {
	l.close()
}

func (l *Live) fetchRoom() error {
	roomInfo, err := l.getRoomInfo()
	if err != nil {
		return err
	}
	l.Info = roomInfo
	//
	// giftInfo, err := l.getGiftInfo()
	// if err != nil {
	//	return err
	// }
	// l.GiftInfo = giftInfo

	err = l.getRoomData()
	if err != nil {
		return err
	}
	return nil
}

// GetRoomInfo will only fetch the room info, normally available with live.Info
//
//	but not start tracking a live stream.
func (t *TikTok) GetRoomInfo(username string) (*RoomInfo, error) {
	id, err := t.getRoomID(username)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch room ID by username")
	}

	l := Live{
		t:  t,
		ID: id,
	}

	roomInfo, err := l.getRoomInfo()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch room info")
	}
	return roomInfo, nil
}

// TrackUser will start to track the livestream of a user, if live.
// To listen to events emitted by the livestream, such as comments and viewer
//
//	count, listen to the Live.Events channel.
//
// It will start a go routine and connect to the tiktok websocket.
func (t *TikTok) TrackUser(username string) (*Live, error) {
	id, err := t.getRoomID(username)
	if err != nil {
		return nil, err
	}

	return t.TrackRoom(id)
}

// TrackRoom will start to track a room by room ID.
// It will start a go routine and connect to the tiktok websocket.
func (t *TikTok) TrackRoom(roomId string) (*Live, error) {
	live := t.newLive(roomId)

	if err := live.fetchRoom(); err != nil {
		close(live.Events)
		return nil, err
	}

	if err := live.connectRoom(); err != nil {
		return nil, err
	}

	return live, nil
}

func (live *Live) connectRoom() error {
	return live.tryConnectionUpgrade()
}

func (t *TikTok) getRoomID(user string) (string, error) {
	userInfo, err := t.GetUserInfo(user)
	if err != nil {
		return "", err
	}

	if userInfo.RoomID == "" {
		return "", ErrUserOffline
	}
	return userInfo.RoomID, nil
}

func (l *Live) getRoomInfo() (*RoomInfo, error) {
	t := l.t

	params := copyMap(defaultGETParams)
	params["room_id"] = l.ID

	body, _, err := t.sendRequest(&reqOptions{
		Endpoint: urlRoomInfo,
		Query:    params,
	}, nil)
	if err != nil {
		return nil, err
	}

	var rsp roomInfoRsp
	if err := json.Unmarshal(body, &rsp); err != nil {
		return nil, err
	}

	if rsp.RoomInfo.Status == 4 {
		return rsp.RoomInfo, ErrLiveHasEnded
	}
	return rsp.RoomInfo, nil
}

func (l *Live) getGiftInfo() (*GiftInfo, error) {
	t := l.t

	params := copyMap(defaultGETParams)
	params["room_id"] = l.ID

	body, _, err := t.sendRequest(&reqOptions{
		Endpoint: urlGiftInfo,
		Query:    params,
	}, nil)
	if err != nil {
		return nil, err
	}

	var rsp giftInfoRsp
	if err := json.Unmarshal(body, &rsp); err != nil {
		return nil, err
	}
	return rsp.GiftInfo, nil
}

func (l *Live) getRoomData() error {
	t := l.t

	params := copyMap(defaultGETParams)
	params["room_id"] = l.ID
	params["device_id"] = getRandomDeviceID()
	if l.cursor != "" {
		params["cursor"] = l.cursor
	}

	body, headers, err := t.sendRequest(&reqOptions{
		Endpoint: urlRoomData,
		Query:    params,
	}, nil)
	if err != nil {
		return err
	}
	ttsCookie := headers.Get("X-Set-TT-Cookie")
	cookies, err := http.ParseCookie(ttsCookie)
	if err != nil {
		return fmt.Errorf("X-SetTT-Cookie not parsable: %w", err)
	}

	for i := range cookies {
		cookies[i].Domain = ".tiktok.com"
	}
	u, err := url.Parse("https://tiktok.com")
	if err != nil {
		return fmt.Errorf("semantica error couldnot parse secure cookie endpoint, please report: %w", err)
	}
	t.c.Jar.SetCookies(u, cookies)

	var rsp pb.WebcastResponse
	if err := proto.Unmarshal(body, &rsp); err != nil {
		return err
	}

	l.cursor = rsp.Cursor
	if rsp.PushServer != "" && rsp.RouteParamsMap != nil {
		l.wsURL = rsp.PushServer
		l.wsParams = make(map[string]string)
		for k, v := range rsp.RouteParamsMap {
			l.wsParams[k] = v
		}

	}

	for _, msg := range rsp.Messages {
		parsed, err := parseMsg(msg, t.warnHandler, t.debugHandler, t.enableExperimentalEvents)
		if err != nil {
			return err
		}
		l.Events <- parsed
	}

	return nil
}

// DownloadStream will download the stream to an .mkv file.
//
// A filename can be optionally provided as an argument, if not provided one
//
//	will be generated, with the stream start time in the format of 2022y05m25dT13h03m16s.
//
// The stream start time can be found in Live.Info.CreateTime as epoch seconds.
func (l *Live) DownloadStream(file ...string) error {
	// Check if ffmpeg is installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return ErrFFMPEGNotFound
	}

	// Get URl
	url := l.Info.StreamURL.HlsPullURL
	if url == "" {
		return ErrURLNotFound
	}

	// Set file path
	var path string
	format := ".mkv"
	if len(file) > 0 {
		path = file[0]
		if !strings.HasSuffix(path, format) {
			path += format
		}
	} else {
		path = fmt.Sprintf("%s-%s%s", l.Info.Owner.Username, time.Unix(l.Info.CreateTime, 0).Format("2006y01m02dT15h04m05s"), format)
	}
	if _, err := os.Stat(path); err == nil {
		t := strings.TrimSuffix(path, format)
		path = fmt.Sprintf("%s-%d%s", t, time.Now().Unix(), format)
	}

	options := []string{}
	if l.t.proxy != nil && (l.t.proxy.Scheme == "http" || l.t.proxy.Scheme == "https") {
		options = append([]string{"-http_proxy", l.t.proxy.String()}, options...)
	}
	options = append(options, []string{"-i", url}...)
	// important to come after the -i
	options = append(options, "-c:v", "libx264", "-c:a", "aac", "-bufsize", "2M", "-fflags", "+discardcorrupt",
		"-fflags", "+genpts", path)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "ffmpeg", options...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	mu := new(sync.Mutex)
	finished := false

	go func(c *exec.Cmd) {
	drainFor:
		for {
			select {
			case <-l.done():
				break drainFor
			// Needs to be drained or deadlocks can occur, maybe at some point make a download option
			// where the library user needs to drain
			case _, ok := <-l.Events:
				if !ok {
					break drainFor
				}
			}
		}
		mu.Lock()
		func() {
			defer mu.Unlock()
			if !finished {
				// Send q key press to quit
				stdin.Write([]byte("q\n"))
				time.Sleep(5 * time.Second)
			}
			cancel()
		}()
		// naturally should be closed, but keep draining to prevent deadlocks
		for {
			if _, ok := <-l.Events; !ok {
				break
			}
		}
	}(cmd)

	// Go routine to wait for process to exit and return result
	l.wg.Add(1)
	go func(c *exec.Cmd, stdout, stderr io.ReadCloser) {
		defer l.wg.Done()
		removeNewlineExp := regexp.MustCompile(`[\r\n]+$`)
		streamWG := new(sync.WaitGroup)
		streamWG.Add(2)
		go func() {
			defer streamWG.Done()
			r := bufio.NewReader(stdout)
			for {
				str, err := r.ReadString('\n')
				if err != nil {
					return
				}
				l.t.debugHandler(removeNewlineExp.ReplaceAllString(str, ""))
			}
		}()

		go func() {
			defer streamWG.Done()
			r := bufio.NewReader(stderr)
			for {
				str, err := r.ReadString('\n')
				if err != nil {
					return
				}
				l.t.debugHandler(removeNewlineExp.ReplaceAllString(str, ""))
			}
		}()
		streamWG.Wait()
		err := cmd.Wait()
		mu.Lock()
		defer mu.Unlock()
		finished = true
		if err != nil {
			l.t.errHandler(fmt.Sprintf("Download for failed: %s", err))
		}
		return
		l.t.infoHandler(fmt.Sprintf("Download for %s finished!", l.Info.Owner.Username))
	}(cmd, stdout, stderr)
	l.wg.Wait()
	l.t.infoHandler(fmt.Sprintf("Started downloading stream by %s to %s\n", l.Info.Owner.Username, path))

	return nil
}

func (t *TikTok) signURL(reqUrl string, options *reqOptions) ([]byte, http.Header, error) {
	query := map[string]string{
		"client":  t.clientName,
		"uuc":     strconv.Itoa(t.streams),
		"url":     reqUrl,
		"room_id": options.Query["room_id"],
	}
	if t.apiKey != "" {
		query["apiKey"] = t.apiKey
	}
	// A badly formed implementation using this library might spam connection requests (ask me
	// how I know) this limiter is a safety guard to never go over the signer's advertised
	// capabilities so the client does not exceed limits or get banned from the signer.
	t.limiter.Take()
	body, header, err := t.sendRequest(&reqOptions{
		URI:      t.signerUrl,
		Endpoint: urlSignReq,
		Query:    query,
	}, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, fmt.Sprintf("Failed to sign request: %s", body))
	}

	return body, header, nil
}

// Only able to get this while logged in
// func (l *Live) GetRankList() (*RankList, error) {
// 	t := l.t
//
// 	params := copyMap(defaultGETParams)
// 	params["room_id"] = l.ID
// 	params["channel"] = "tiktok_web"
// 	params["anchor_id"] = "idk"
//
// 	body, err := t.sendRequest(&reqOptions{
// 		Endpoint: urlRankList,
// 		Query:    params,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	var rsp rankListRsp
// 	if err := json.Unmarshal(body, &rsp); err != nil {
// 		return nil, err
// 	}
//
// 	return &rsp.RankList, nil
// }
