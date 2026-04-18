package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// Transport owns the JSON-RPC over LSP-framing wire format: each
// message is `Content-Length: N\r\n\r\n<N-byte JSON body>`. The
// transport is a thin wrapper around bufio.Reader / io.Writer;
// concurrent writes are serialised so response and notification
// ordering is stable.
type Transport struct {
	r    *bufio.Reader
	w    io.Writer
	wmu  sync.Mutex
}

// NewTransport wires the transport to an input reader and output
// writer — typically os.Stdin / os.Stdout, but in-process tests
// use bytes.Buffer pairs.
func NewTransport(r io.Reader, w io.Writer) *Transport {
	return &Transport{r: bufio.NewReader(r), w: w}
}

// ReadMessage consumes one LSP-framed message and returns the
// raw body. A malformed frame (missing Content-Length, non-integer
// length, body shorter than declared) produces a protocol error
// the caller can route to JSON-RPC error code ErrParse.
func (t *Transport) ReadMessage() ([]byte, error) {
	length := -1
	for {
		line, err := t.r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trim := strings.TrimRight(line, "\r\n")
		if trim == "" {
			break // end of header block
		}
		name, value, ok := splitHeader(trim)
		if !ok {
			return nil, fmt.Errorf("malformed header %q", trim)
		}
		if strings.EqualFold(name, "Content-Length") {
			n, parseErr := strconv.Atoi(strings.TrimSpace(value))
			if parseErr != nil || n < 0 {
				return nil, fmt.Errorf("invalid Content-Length %q", value)
			}
			length = n
		}
	}
	if length < 0 {
		return nil, errors.New("missing Content-Length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(t.r, body); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}

// WriteMessage frames and writes a JSON body with the canonical
// `Content-Length: N\r\n\r\n<body>` envelope. Concurrent callers
// are serialised — an LSP response MUST NOT interleave with a
// notification on the same stream.
func (t *Transport) WriteMessage(body []byte) error {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := t.w.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := t.w.Write(body); err != nil {
		return err
	}
	return nil
}

// WriteResponse marshals resp and sends it through the transport.
func (t *Transport) WriteResponse(resp Response) error {
	buf, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	return t.WriteMessage(buf)
}

// WriteNotification marshals n and sends it through the transport.
func (t *Transport) WriteNotification(n Notification) error {
	buf, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}
	return t.WriteMessage(buf)
}

// splitHeader is an RFC-822 style `Name: Value` split. Accepts
// both the LSP-canonical `:` separator and the slightly-looser
// trimmed variant some clients send.
func splitHeader(line string) (name, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}
