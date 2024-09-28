package gotiktoklive

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	pb "github.com/steampoweredtaco/gotiktoklive/proto"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"golang.org/x/net/proxy"
	"google.golang.org/protobuf/proto"
)

func (l *Live) connect(addr string, params map[string]string) error {
	u, err := url.Parse("https://tiktok.com/")
	if err != nil {
		return nil
	}

	// Take the cookies from the HTTP client cookie jar and pass them as a GET parameter.
	cookies := l.t.c.Jar.Cookies(u)
	headers := http.Header{}
	var s string
	for _, cookie := range cookies {
		if s == "" {
			s = cookie.String()
		} else {
			s += "; " + cookie.String()
		}
	}
	if len(cookies) > 0 {
		headers.Set("Cookie", s)
	}

	vs := url.Values{}
	for key, value := range defaultGETParams {
		vs.Add(key, value)
	}
	for key, value := range params {
		vs.Add(key, value)
	}
	vs.Add("room_id", l.ID)

	wsURL := fmt.Sprintf("%s?%s", addr, vs.Encode())

	// Read proxies from HTTP env variables, HTTPS takes precedent
	var proxyURI *url.URL
	envVars := []string{"HTTP_PROXY", "HTTPS_PROXY"}
	for _, envVar := range envVars {
		if httpProxy := os.Getenv(envVar); httpProxy != "" {
			uri, err := url.Parse(httpProxy)
			if err != nil {
				l.t.warnHandler(fmt.Errorf("Failed to parse %s url: %w", envVar, err))
			} else {
				proxyURI = uri
			}
		}
	}

	// If proxy manually defined, use that
	if l.t.proxy != nil {
		proxyURI = l.t.proxy
	}

	// Default proxy net dial
	proxyNetDial := proxy.FromEnvironment()

	// If custom URI is defined, genereate proxy net dial from that
	if proxyURI != nil {
		dial, err := proxy.FromURL(proxyURI, Direct)
		if err != nil {
			l.t.warnHandler(fmt.Errorf("Failed to configure proxy dialer: %w", err))
		} else {
			proxyNetDial = dial
		}
	}

	dialer := ws.Dialer{
		Header: ws.HandshakeHeaderHTTP(headers),
		NetDial: func(ctx context.Context, a, b string) (net.Conn, error) {
			return proxyNetDial.Dial(a, b)
		},
		// NetDial:   proxy.Dial,
		Protocols: []string{"echo-protocol"},
	}
	conn, _, _, err := dialer.Dial(context.Background(), wsURL)
	if err != nil {
		return fmt.Errorf("Failed to connect: %w", err)
	}
	l.wss = conn
	return nil
}

func (l *Live) readSocket() {
	defer l.wss.Close()
	defer func() {
		select {
		case <-time.After(5 * time.Second):
		case l.Events <- DisconnectEvent{}:
		}
	}()
	defer l.cancel()

	want := ws.OpBinary
	s := ws.StateClientSide

	controlHandler := wsutil.ControlFrameHandler(l.wss, s)
	rd := wsutil.Reader{
		Source:          l.wss,
		State:           s,
		CheckUTF8:       true,
		SkipHeaderCheck: false,
		OnIntermediate:  controlHandler,
	}

	for {
		hdr, err := rd.NextFrame()
		if err != nil {
			l.t.errHandler(fmt.Errorf("failed to read websocket from server: %w", err))
			l.wss.Close()
			return
		}
		// If msg is ping or close
		if hdr.OpCode.IsControl() {
			if err := controlHandler(hdr, &rd); err != nil {
				l.t.errHandler(fmt.Errorf("websocket control handler failed: %w", err))
			}
			continue
		}

		if hdr.OpCode == ws.OpClose {
			l.t.warnHandler("Websocket connection was closed by server.")
			if l.t.enableWSTrace {
				l.t.wsTraceChan <- struct{ direction, hex string }{direction: "<=", hex: "websocket closed"}
			}
			return
		}

		// Wrong OpCode
		if hdr.OpCode&want == 0 {
			msgBytes, err := io.ReadAll(&rd)
			if err != nil {
				l.t.errHandler(fmt.Errorf("Failed to read websocket message: %w", err))
			}
			if l.t.enableWSTrace {
				l.t.wsTraceChan <- struct{ direction, hex string }{direction: fmt.Sprintf("<= (unexpected opcode %02x)", hdr.OpCode), hex: hex.EncodeToString(msgBytes)}
			}

			continue
		}

		// Read message
		msgBytes, err := io.ReadAll(&rd)
		if l.t.enableWSTrace {
			l.t.wsTraceChan <- struct{ direction, hex string }{direction: "<=", hex: hex.EncodeToString(msgBytes)}
		}

		if err != nil {
			l.t.errHandler(fmt.Errorf("Failed to read websocket message: %w", err))
		}

		if err := l.parseWssMsg(msgBytes); err != nil {
			l.t.errHandler(fmt.Errorf("Failed to parse websocket message: %w", err))
		}

		// Gracefully shutdown
		select {
		case <-l.done():
			return
		case <-l.t.done():
			l.t.infoHandler("Close websocket, global context done")
			return
		default:
		}
	}
}

func (l *Live) parseWssMsg(wssMsg []byte) error {
	var rsp pb.WebcastPushFrame
	if err := proto.Unmarshal(wssMsg, &rsp); err != nil {
		return fmt.Errorf("Failed to unmarshal proto WebcastWebsocketMessage: %w", err)
	}

	if rsp.PayloadType == "msg" {
		var response pb.WebcastResponse
		if err := proto.Unmarshal(rsp.Payload, &response); err != nil {
			return fmt.Errorf("Failed to unmarshal proto WebcastResponse: %w", err)
		}
		if response.NeedsAck {
			if err := l.sendAck(rsp.LogId, response.InternalExt); err != nil {
				// Might as well finishing processing all messages, the connection reset will be
				// caught later
				l.t.warnHandler("Failed to send websocket ack msg, likely connection reset: %w", err)
			}
		}
		l.cursor = response.Cursor

		if l.t.Debug {
			l.t.debugHandler(fmt.Sprintf("Got %d messages, %s", len(response.Messages), response.Cursor))
		}

		for _, rawMsg := range response.Messages {
			msg, err := parseMsg(rawMsg, l.t.warnHandler, l.t.debugHandler, l.t.enableExperimentalEvents)
			if err != nil {
				return fmt.Errorf("Failed to parse response message: %w", err)
			}
			if msg != nil {
				// TODO, let's fix this
				// If channel is full, discard the first message
				if len(l.Events) == l.chanSize {
					<-l.Events
				}
				l.Events <- msg
			}

			// If livestream has ended
			if m, ok := msg.(ControlEvent); ok &&
				(pb.ControlAction(m.Action) == pb.ControlAction_STREAM_ENDED ||
					pb.ControlAction(m.Action) == pb.ControlAction_STREAM_ENDED_BAN) {
				l.t.warnHandler(fmt.Sprint("live has ended due to %s, closing event stream", pb.ControlAction(m.Action).String()))
				l.cancel()
			}
		}
		return nil
	}
	if l.t.Debug {
		l.t.debugHandler(fmt.Sprintf("Message type unknown, %s : '%s\n%s", rsp.PayloadType, string(rsp.Payload), hex.EncodeToString(wssMsg)))
	}
	return nil
}

func (l *Live) sendPing() {
	const helloHex = "3a026862"
	b, err := hex.DecodeString(helloHex)
	if err != nil {
		l.t.errHandler(err)
	}

	t := time.NewTicker(10000 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case <-l.done():
			return
		case <-l.t.done():
			return
		case <-t.C:
			if err := wsutil.WriteClientBinary(l.wss, b); err != nil {
				l.t.errHandler(fmt.Errorf("Failed to send ping: %w", err))
			} else {
				if l.t.enableWSTrace {
					l.t.wsTraceChan <- struct{ direction, hex string }{direction: "=>", hex: helloHex}
				}
			}
		}
	}
}

func (l *Live) sendAck(id uint64, extra []byte) error {
	msg := pb.WebcastPushFrame{
		LogId:       id,
		PayloadType: "ack",
		Payload:     extra,
	}

	b, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	if err := wsutil.WriteClientBinary(l.wss, b); err != nil {
		return err
	}
	if l.t.enableWSTrace {
		l.t.wsTraceChan <- struct{ direction, hex string }{direction: "=>", hex: hex.EncodeToString(b)}
	}
	return nil
}

func (l *Live) tryConnectionUpgrade() error {
	if l.wsURL == "" {
		return fmt.Errorf("cannot upgrade connection without a wsURL")
	}
	if l.wsParams == nil {
		return fmt.Errorf("cannot upgrade connection without a wsURL")
	}
	err := l.connect(l.wsURL, l.wsParams)
	if err != nil {
		close(l.Events)
		return fmt.Errorf("Connection upgrade failed: %w", err)
	}
	if l.t.Debug {
		l.t.debugHandler("Connected to websocket")
	}

	l.wg.Add(2)
	go func() {
		defer l.wg.Done()
		defer close(l.Events)
		l.readSocket()
	}()
	go func() {
		defer l.wg.Done()
		l.sendPing()
	}()

	l.t.infoHandler("Connected to websocket")
	return nil
}
