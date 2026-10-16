package system

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// loopbackConn is a net.Conn over two buffers, so frame handling can be driven
// without a socket and without a peer that might block on a write. Only Read,
// Write and Close are meaningful; the embedded interface satisfies the rest of
// the method set, which wsConn never touches.
type loopbackConn struct {
	net.Conn
	reads  *bytes.Reader
	writes bytes.Buffer
}

func newLoopbackConn(input []byte) *loopbackConn {
	return &loopbackConn{reads: bytes.NewReader(input)}
}

func (c *loopbackConn) Read(p []byte) (int, error)  { return c.reads.Read(p) }
func (c *loopbackConn) Write(p []byte) (int, error) { return c.writes.Write(p) }
func (c *loopbackConn) Close() error                { return nil }

// newTestWSConn returns a connection whose frames are read from data and whose
// outgoing frames land in the returned buffer.
func newTestWSConn(data []byte) (*wsConn, *bytes.Buffer) {
	conn := newLoopbackConn(data)
	return &wsConn{conn: conn, br: bufio.NewReader(conn)}, &conn.writes
}

// rawFrame builds a frame byte for byte, deliberately not through
// wsConn.writeFrame: the reader has to be tested against the wire format rather
// than against the writer's idea of it. A nil mask produces an unmasked frame,
// which is what a server sends.
func rawFrame(fin bool, opcode byte, mask *[4]byte, payload []byte) []byte {
	out := make([]byte, 0, len(payload)+14)

	first := opcode
	if fin {
		first |= 0x80
	}
	out = append(out, first)

	var maskBit byte
	if mask != nil {
		maskBit = 0x80
	}
	switch n := len(payload); {
	case n < 126:
		out = append(out, maskBit|byte(n))
	case n <= 0xffff:
		out = append(out, maskBit|126, byte(n>>8), byte(n))
	default:
		out = append(out, maskBit|127)
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], uint64(n))
		out = append(out, ext[:]...)
	}

	if mask == nil {
		return append(out, payload...)
	}
	out = append(out, mask[:]...)
	for i, b := range payload {
		out = append(out, b^mask[i%4])
	}
	return out
}

// decodeClientFrame parses a client-to-server frame per RFC 6455 and returns the
// unmasked payload, so a written frame can be checked without reusing the reader
// under test.
func decodeClientFrame(t *testing.T, frame []byte) (fin bool, opcode byte, payload []byte) {
	t.Helper()

	if len(frame) < 2 {
		t.Fatalf("frame is %d bytes, too short to hold a header", len(frame))
	}
	fin = frame[0]&0x80 != 0
	opcode = frame[0] & 0x0f

	if frame[1]&0x80 == 0 {
		t.Fatalf("client frame is unmasked, which RFC 6455 §5.3 forbids")
	}

	offset := 2
	length := uint64(frame[1] & 0x7f)
	switch length {
	case 126:
		length = uint64(binary.BigEndian.Uint16(frame[offset : offset+2]))
		offset += 2
	case 127:
		length = binary.BigEndian.Uint64(frame[offset : offset+8])
		offset += 8
	}

	var mask [4]byte
	copy(mask[:], frame[offset:offset+4])
	offset += 4

	if uint64(len(frame)-offset) != length {
		t.Fatalf("frame carries %d payload bytes, the header says %d", len(frame)-offset, length)
	}
	payload = make([]byte, length)
	for i := range payload {
		payload[i] = frame[offset+i] ^ mask[i%4]
	}
	return fin, opcode, payload
}

// TestWSAcceptKey checks the one value RFC 6455 pins down with an example, from
// the handshake in §1.3.
func TestWSAcceptKey(t *testing.T) {
	const (
		key  = "dGhlIHNhbXBsZSBub25jZQ=="
		want = "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	)
	if got := wsAcceptKey(key); got != want {
		t.Errorf("wsAcceptKey(%q) = %q, want %q", key, got, want)
	}
}

// TestWSReadFrameLengths covers the three logical lengths a frame can encode and
// both masking states, since the length switch and the unmask loop are separate
// pieces of logic.
func TestWSReadFrameLengths(t *testing.T) {
	mask := &[4]byte{0xa1, 0xb2, 0xc3, 0xd4}

	tests := []struct {
		name   string
		size   int
		masked bool
		marker byte
	}{
		{"empty", 0, false, 0},
		{"masked, one byte", 1, true, 1},
		{"seven-bit length", 125, false, 125},
		{"sixteen-bit length boundary", 126, true, 126},
		{"sixteen-bit length", 65535, false, 126},
		{"sixty-four-bit length", 65536, true, 127},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := bytes.Repeat([]byte("z"), tc.size)
			var m *[4]byte
			if tc.masked {
				m = mask
			}
			frame := rawFrame(true, wsOpBinary, m, payload)

			// The marker nibble is the reader's other half: a wrong choice here
			// means the next two or eight bytes are read as payload.
			if got := frame[1] & 0x7f; got != tc.marker {
				t.Fatalf("frame encodes length marker %d, want %d", got, tc.marker)
			}

			conn, _ := newTestWSConn(frame)
			fin, opcode, got, err := conn.readFrame()
			if err != nil {
				t.Fatalf("readFrame: %v", err)
			}
			if !fin || opcode != wsOpBinary {
				t.Errorf("readFrame returned fin=%v opcode=%#x, want fin=true opcode=%#x", fin, opcode, wsOpBinary)
			}
			if !bytes.Equal(got, payload) {
				t.Errorf("readFrame returned %d payload bytes, want %d", len(got), len(payload))
			}
		})
	}
}

func TestWSReadMessageReassemblesFragments(t *testing.T) {
	data := rawFrame(false, wsOpText, nil, []byte("hel"))
	data = append(data, rawFrame(false, wsOpContinuation, nil, []byte("lo "))...)
	data = append(data, rawFrame(true, wsOpContinuation, nil, []byte("world"))...)

	conn, _ := newTestWSConn(data)
	opcode, payload, err := conn.readMessage()
	if err != nil {
		t.Fatalf("readMessage: %v", err)
	}
	if opcode != wsOpText {
		t.Errorf("opcode = %#x, want %#x (the opcode the message opened with)", opcode, wsOpText)
	}
	if string(payload) != "hello world" {
		t.Errorf("payload = %q, want %q", payload, "hello world")
	}
}

// TestWSReadMessageAnswersPing places a ping in the middle of a fragmented
// message, which is where a careless client either drops the partial message or
// fails to answer. Control frames are never fragmented and never join the
// message being reassembled.
func TestWSReadMessageAnswersPing(t *testing.T) {
	data := rawFrame(false, wsOpText, nil, []byte("hel"))
	data = append(data, rawFrame(true, wsOpPing, nil, []byte("are you there"))...)
	data = append(data, rawFrame(true, wsOpContinuation, nil, []byte("lo"))...)

	conn, written := newTestWSConn(data)
	opcode, payload, err := conn.readMessage()
	if err != nil {
		t.Fatalf("readMessage: %v", err)
	}
	if opcode != wsOpText || string(payload) != "hello" {
		t.Errorf("readMessage returned opcode=%#x payload=%q, want opcode=%#x payload=%q", opcode, payload, wsOpText, "hello")
	}

	fin, pongOpcode, pongPayload := decodeClientFrame(t, written.Bytes())
	if !fin || pongOpcode != wsOpPong {
		t.Errorf("reply was fin=%v opcode=%#x, want fin=true opcode=%#x", fin, pongOpcode, wsOpPong)
	}
	if string(pongPayload) != "are you there" {
		t.Errorf("pong payload = %q, want %q", pongPayload, "are you there")
	}
}

func TestWSReadMessageTreatsCloseAsEOF(t *testing.T) {
	conn, _ := newTestWSConn(rawFrame(true, wsOpClose, nil, nil))
	if _, _, err := conn.readMessage(); err == nil {
		t.Error("readMessage on a close frame returned no error, want EOF")
	}
}

func TestWSReadFrameRejects(t *testing.T) {
	tests := []struct {
		name  string
		frame []byte
	}{
		{
			// A reserved bit set with nothing negotiated that could justify it.
			name:  "reserved bit",
			frame: []byte{0x80 | 0x40 | wsOpText, 0x00},
		},
		{
			// A sixty-four-bit length far past the message limit. The reader must
			// refuse before allocating, which is the point of the guard.
			name:  "oversized frame",
			frame: []byte{0x80 | wsOpText, 0x80 | 127, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00},
		},
		{
			name:  "truncated header",
			frame: []byte{0x80 | wsOpText},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, _ := newTestWSConn(tc.frame)
			if _, _, _, err := conn.readFrame(); err == nil {
				t.Error("readFrame accepted a frame it should have rejected")
			}
		})
	}
}

func TestWSReadMessageRejectsMalformedSequences(t *testing.T) {
	tests := []struct {
		name  string
		frame []byte
	}{
		{
			name:  "continuation with no message in progress",
			frame: rawFrame(true, wsOpContinuation, nil, []byte("orphan")),
		},
		{
			name: "second data frame before the first finished",
			frame: append(
				rawFrame(false, wsOpText, nil, []byte("a")),
				rawFrame(true, wsOpText, nil, []byte("b"))...,
			),
		},
		{
			name:  "opcode that does not exist",
			frame: rawFrame(true, 0x3, nil, []byte("x")),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, _ := newTestWSConn(tc.frame)
			if _, _, err := conn.readMessage(); err == nil {
				t.Error("readMessage accepted a frame sequence it should have rejected")
			}
		})
	}
}

// TestWSWriteFrameUsesShortestLengthEncoding checks the encoding choice, not
// just the round trip: a frame that used the sixty-four-bit form for a short
// payload would still decode, and still be wrong.
func TestWSWriteFrameUsesShortestLengthEncoding(t *testing.T) {
	tests := []struct {
		size   int
		marker byte
	}{
		{0, 0},
		{125, 125},
		{126, 126},
		{65535, 126},
		{65536, 127},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d bytes", tc.size), func(t *testing.T) {
			payload := bytes.Repeat([]byte("y"), tc.size)

			conn, written := newTestWSConn(nil)
			if err := conn.writeFrame(wsOpBinary, payload, true); err != nil {
				t.Fatalf("writeFrame: %v", err)
			}

			frame := written.Bytes()
			if got := frame[1] & 0x7f; got != tc.marker {
				t.Errorf("frame encodes length marker %d, want %d", got, tc.marker)
			}

			fin, opcode, got := decodeClientFrame(t, frame)
			if !fin || opcode != wsOpBinary {
				t.Errorf("frame parses as fin=%v opcode=%#x, want fin=true opcode=%#x", fin, opcode, wsOpBinary)
			}
			if !bytes.Equal(got, payload) {
				t.Errorf("frame parses back to %d payload bytes, want %d", len(got), len(payload))
			}
		})
	}
}

// wsStubRequest is what the stub server saw, handed to the test over a channel
// so there is no race on the fields.
type wsStubRequest struct {
	RequestLine string
	Headers     map[string]string
}

// wsStubServer accepts one connection, answers the handshake with the given
// status and raw header lines, and returns the URL to dial along with a channel
// carrying the request it received.
//
// The listen socket is closed when the test finishes, so a test that expects the
// dial to fail does not hang and does not leak.
func wsStubServer(t *testing.T, status string, headerLines []string) (string, <-chan wsStubRequest) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	requests := make(chan wsStubRequest, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		br := bufio.NewReader(conn)
		request := wsStubRequest{Headers: map[string]string{}}

		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		request.RequestLine = strings.TrimRight(line, "\r\n")

		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			name, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			request.Headers[strings.ToLower(strings.TrimSpace(name))] = strings.TrimSpace(value)
		}
		requests <- request

		var response strings.Builder
		fmt.Fprintf(&response, "HTTP/1.1 %s\r\n", status)

		// The accept value can only be computed once the key is known, so it is
		// added here unless the caller supplied one (the bad-accept case).
		suppliedAccept := false
		for _, header := range headerLines {
			name, _, ok := strings.Cut(header, ":")
			if ok && strings.EqualFold(strings.TrimSpace(name), "Sec-WebSocket-Accept") {
				suppliedAccept = true
			}
		}
		if strings.HasPrefix(status, "101") && !suppliedAccept {
			fmt.Fprintf(&response, "Sec-WebSocket-Accept: %s\r\n", wsAcceptKey(request.Headers["sec-websocket-key"]))
		}
		for _, header := range headerLines {
			response.WriteString(header + "\r\n")
		}
		response.WriteString("\r\n")

		conn.Write([]byte(response.String()))
	}()

	return "ws://" + ln.Addr().String(), requests
}

func TestWSDialWebSocketSendsAValidHandshake(t *testing.T) {
	url, requests := wsStubServer(t, "101 Switching Protocols", []string{
		"Upgrade: websocket",
		"Connection: Upgrade",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := dialWebSocket(ctx, url)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	defer conn.close()

	request := <-requests
	if request.RequestLine != "GET / HTTP/1.1" {
		t.Errorf("request line = %q, want %q", request.RequestLine, "GET / HTTP/1.1")
	}

	for name, want := range map[string]string{
		"upgrade":               "websocket",
		"connection":            "Upgrade",
		"sec-websocket-version": "13",
	} {
		if got := request.Headers[name]; !strings.EqualFold(got, want) {
			t.Errorf("%s header = %q, want %q", name, got, want)
		}
	}

	key := request.Headers["sec-websocket-key"]
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Fatalf("Sec-WebSocket-Key %q is not base64: %v", key, err)
	}
	if len(decoded) != 16 {
		t.Errorf("Sec-WebSocket-Key decodes to %d bytes, want 16", len(decoded))
	}
}

// TestWSDialWebSocketAcceptsOddHeaderCasing covers a real difference between
// servers: HTTP header names are case-insensitive, so the response parser must
// not depend on a relay writing them the way our request did.
func TestWSDialWebSocketAcceptsOddHeaderCasing(t *testing.T) {
	url, _ := wsStubServer(t, "101 Switching Protocols", []string{
		"UPGRADE: WebSocket",
		"connection: upgrade",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := dialWebSocket(ctx, url)
	if err != nil {
		t.Fatalf("dialWebSocket: %v", err)
	}
	conn.close()
}

func TestWSDialWebSocketRejects(t *testing.T) {
	tests := []struct {
		name        string
		status      string
		headerLines []string
	}{
		{
			name:   "refused upgrade",
			status: "403 Forbidden",
		},
		{
			name:   "no upgrade confirmation",
			status: "101 Switching Protocols",
		},
		{
			name:        "wrong accept value",
			status:      "101 Switching Protocols",
			headerLines: []string{"Upgrade: websocket", "Sec-WebSocket-Accept: c2V2ZXJhbCBsaWVz"},
		},
		{
			// Nothing was offered, so a peer claiming to have enabled an
			// extension will send frames we would try to read as JSON.
			name:        "negotiated an extension",
			status:      "101 Switching Protocols",
			headerLines: []string{"Upgrade: websocket", "Sec-WebSocket-Extensions: permessage-deflate"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			url, _ := wsStubServer(t, tc.status, tc.headerLines)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn, err := dialWebSocket(ctx, url)
			if err == nil {
				conn.close()
				t.Fatal("dialWebSocket accepted a handshake it should have rejected")
			}
		})
	}
}

func TestWSDialWebSocketRejectsUnsupportedScheme(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, err := dialWebSocket(ctx, "ftp://relay.example"); err == nil {
		t.Error("dialWebSocket accepted a non-websocket scheme")
	}
}
