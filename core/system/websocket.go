package system

// A minimal RFC 6455 client, sized for exactly one job: ask a Nostr relay for
// one account's kind-0 event and hang up.
//
// Hand-rolled because the alternative is a module dependency carried by every
// build for one feature, and because a relay's server side is the simple half of
// the protocol: no extension is negotiated, every message is small, and the
// conversation is strictly REQ → EVENT → EOSE → CLOSE.
//
// What is deliberately NOT simplified away is the part a hand-rolled client gets
// wrong in the field rather than in a test: client frames are masked, fragmented
// messages are reassembled, interleaved pings are answered, and a server that
// asks for permessage-deflate is refused outright instead of being handed bytes
// it would try to parse as JSON.

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
)

const (
	wsOpContinuation = 0x0
	wsOpText         = 0x1
	wsOpBinary       = 0x2
	wsOpClose        = 0x8
	wsOpPing         = 0x9
	wsOpPong         = 0xa
)

// wsGUID is the constant RFC 6455 §4.2.2 concatenates with the client key to
// form the accept value.
const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// wsMaxFrame caps one frame and one reassembled message. A kind-0 event with a
// generous `about` is a few kilobytes; past this the peer is broken or hostile,
// and reading it into memory is the only way to find out.
const wsMaxFrame = 1 << 20

type wsConn struct {
	conn net.Conn
	br   *bufio.Reader
}

// wsAcceptKey computes the Sec-WebSocket-Accept value for a client key.
//
// SHA-1 is mandated by RFC 6455 for this one value. It is a handshake
// handshake nonce, not a signature — nothing is authenticated by it — so its
// collision resistance is irrelevant here, and no other hash would be accepted
// by the peer.
func wsAcceptKey(key string) string {
	sum := sha1.Sum([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// dialWebSocket performs the opening handshake and returns a connection ready
// for frames. The context governs the connection's deadline for its whole life,
// so callers must pass one with a deadline.
func dialWebSocket(ctx context.Context, rawURL string) (*wsConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse relay url %q: %w", rawURL, err)
	}

	var secure bool
	switch u.Scheme {
	case "wss", "https":
		secure = true
	case "ws", "http":
		secure = false
	default:
		return nil, fmt.Errorf("unsupported websocket scheme %q", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("relay url %q has no host", rawURL)
	}
	port := u.Port()
	if port == "" {
		if secure {
			port = "443"
		} else {
			port = "80"
		}
	}

	dialer := &net.Dialer{}
	var conn net.Conn
	if secure {
		conn, err = (&tls.Dialer{
			NetDialer: dialer,
			Config:    &tls.Config{ServerName: host},
		}).DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	}
	if err != nil {
		return nil, err
	}

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		conn.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)

	path := u.Path
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}

	var req strings.Builder
	req.WriteString("GET " + path + " HTTP/1.1\r\n")
	req.WriteString("Host: " + u.Host + "\r\n")
	req.WriteString("Upgrade: websocket\r\n")
	req.WriteString("Connection: Upgrade\r\n")
	req.WriteString("Sec-WebSocket-Key: " + key + "\r\n")
	req.WriteString("Sec-WebSocket-Version: 13\r\n")
	req.WriteString("\r\n")

	if _, err := io.WriteString(conn, req.String()); err != nil {
		conn.Close()
		return nil, err
	}

	// The response is read by hand rather than with http.ReadResponse, which
	// would take the 101 to mean "switching protocols" and hand back a body
	// reader positioned on the WebSocket frames themselves.
	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("read handshake response: %w", err)
	}
	if parts := strings.SplitN(strings.TrimRight(statusLine, "\r\n"), " ", 3); len(parts) < 2 || parts[1] != "101" {
		conn.Close()
		return nil, fmt.Errorf("relay refused the websocket upgrade: %s", strings.TrimSpace(statusLine))
	}

	headers := map[string]string{}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("read handshake headers: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		headers[strings.ToLower(strings.TrimSpace(name))] = strings.TrimSpace(value)
	}

	if got, want := headers["sec-websocket-accept"], wsAcceptKey(key); got != want {
		conn.Close()
		return nil, fmt.Errorf("relay answered with Sec-WebSocket-Accept %q, want %q", got, want)
	}
	if !strings.EqualFold(headers["upgrade"], "websocket") {
		conn.Close()
		return nil, fmt.Errorf("relay did not confirm the websocket upgrade")
	}
	// We offer no extensions. A server that claims to have enabled one will send
	// frames we would try to read as JSON, so refuse rather than decode garbage.
	if ext := headers["sec-websocket-extensions"]; ext != "" {
		conn.Close()
		return nil, fmt.Errorf("relay negotiated the unsupported websocket extension %q", ext)
	}

	return &wsConn{conn: conn, br: br}, nil
}

func (c *wsConn) close() {
	// Best effort: a close frame is polite, and a relay that already went away
	// would just make the write fail.
	c.writeFrame(wsOpClose, nil, true)
	c.conn.Close()
}

// writeFrame writes one masked frame. Every client-to-server frame must be
// masked, and the length uses the shortest of the three encodings.
func (c *wsConn) writeFrame(opcode byte, payload []byte, fin bool) error {
	header := make([]byte, 0, 14)
	first := opcode
	if fin {
		first |= 0x80
	}
	header = append(header, first)

	switch n := len(payload); {
	case n < 126:
		header = append(header, 0x80|byte(n))
	case n <= 0xffff:
		header = append(header, 0x80|126, byte(n>>8), byte(n))
	default:
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], uint64(n))
		header = append(header, 0x80|127)
		header = append(header, ext[:]...)
	}

	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return err
	}
	header = append(header, mask[:]...)

	frame := make([]byte, 0, len(header)+len(payload))
	frame = append(frame, header...)
	for i, b := range payload {
		frame = append(frame, b^mask[i%4])
	}

	_, err := c.conn.Write(frame)
	return err
}

func (c *wsConn) writeText(payload []byte) error {
	return c.writeFrame(wsOpText, payload, true)
}

// readFrame reads one frame, unmasking the payload if the peer masked it.
func (c *wsConn) readFrame() (fin bool, opcode byte, payload []byte, err error) {
	var head [2]byte
	if _, err := io.ReadFull(c.br, head[:]); err != nil {
		return false, 0, nil, err
	}

	fin = head[0]&0x80 != 0
	if head[0]&0x70 != 0 {
		// Nothing was negotiated that could set these, so a peer setting them
		// means it is speaking a protocol we did not agree to.
		return false, 0, nil, fmt.Errorf("websocket frame has reserved bits set")
	}
	opcode = head[0] & 0x0f

	masked := head[1]&0x80 != 0
	length := uint64(head[1] & 0x7f)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(c.br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(c.br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	if length > wsMaxFrame {
		return false, 0, nil, fmt.Errorf("websocket frame of %d bytes exceeds the %d byte limit", length, wsMaxFrame)
	}

	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(c.br, mask[:]); err != nil {
			return false, 0, nil, err
		}
	}

	payload = make([]byte, length)
	if _, err := io.ReadFull(c.br, payload); err != nil {
		return false, 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return fin, opcode, payload, nil
}

// readMessage returns the next complete data message, answering control frames
// as they interleave. The returned opcode is the one the message opened with.
func (c *wsConn) readMessage() (byte, []byte, error) {
	var (
		message []byte
		opcode  byte
	)

	for {
		fin, op, payload, err := c.readFrame()
		if err != nil {
			return 0, nil, err
		}

		// Control frames may arrive between the fragments of a data message and
		// are always final, so they are handled before any bookkeeping — and
		// never touch the message being reassembled.
		switch op {
		case wsOpPing:
			if err := c.writeFrame(wsOpPong, payload, true); err != nil {
				return 0, nil, err
			}
			continue
		case wsOpPong:
			continue
		case wsOpClose:
			c.writeFrame(wsOpClose, nil, true)
			return 0, nil, io.EOF
		}

		switch op {
		case wsOpContinuation:
			if opcode == 0 {
				return 0, nil, fmt.Errorf("websocket continuation frame with no message in progress")
			}
		case wsOpText, wsOpBinary:
			if opcode != 0 {
				return 0, nil, fmt.Errorf("websocket data frame started before the previous message finished")
			}
			opcode = op
		default:
			return 0, nil, fmt.Errorf("unsupported websocket opcode %#x", op)
		}

		if len(message)+len(payload) > wsMaxFrame {
			return 0, nil, fmt.Errorf("websocket message exceeds the %d byte limit", wsMaxFrame)
		}
		message = append(message, payload...)

		if fin {
			return opcode, message, nil
		}
	}
}
