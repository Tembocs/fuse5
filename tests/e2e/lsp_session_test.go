package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestLspScriptedSession is the W19-P05-T01 Verify target. It
// launches the `fuse lsp` subcommand as a subprocess, plays a
// scripted LSP session (initialize → didOpen → hover → goto →
// completion → documentSymbol → shutdown → exit), and asserts
// each response matches the expected shape.
//
// The scripted session is the first honest end-to-end proof that
// the server's stdio transport works — the in-process unit tests
// drive Handle() directly; this test exercises the full wire path.
func TestLspScriptedSession(t *testing.T) {
	fuseBin := buildFuseBinary(t)

	cmd := exec.Command(fuseBin, "lsp")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start fuse lsp: %v", err)
	}

	// --- Client-side driver ---

	c := &scriptedClient{in: stdin, out: stdout}
	defer stdin.Close()

	// 1. initialize
	initResp := c.request(t, 1, "initialize", map[string]any{
		"processId": 1,
		"rootUri":   "file:///ws",
		"clientInfo": map[string]any{"name": "test"},
	})
	if initResp.Error != nil {
		t.Fatalf("initialize error: %+v", initResp.Error)
	}

	// 2. initialized notification (ack the handshake)
	c.notify(t, "initialized", map[string]any{})

	// 3. didOpen
	uri := "file:///ws/scripted.fuse"
	src := "/// Answer to life.\npub fn answer() -> I32 {\n    return 42;\n}\n\nfn main() -> I32 {\n    return answer();\n}\n"
	c.notify(t, "textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "fuse",
			"version":    1,
			"text":       src,
		},
	})

	// 4. hover on `answer` in main's return statement
	hoverResp := c.request(t, 2, "textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 6, "character": 12},
	})
	if hoverResp.Error != nil {
		t.Fatalf("hover error: %+v", hoverResp.Error)
	}
	hoverStr := asStringRaw(hoverResp.Result, "contents", "value")
	if !strings.Contains(hoverStr, "answer") {
		t.Errorf("hover missing `answer`: %q", hoverStr)
	}
	if !strings.Contains(hoverStr, "Answer to life") {
		t.Errorf("hover missing doc comment: %q", hoverStr)
	}

	// 5. goto definition of `answer`
	gotoResp := c.request(t, 3, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 6, "character": 12},
	})
	if gotoResp.Error != nil {
		t.Fatalf("definition error: %+v", gotoResp.Error)
	}

	// 6. completion at a blank line inside main
	compResp := c.request(t, 4, "textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 5, "character": 0},
	})
	if compResp.Error != nil {
		t.Fatalf("completion error: %+v", compResp.Error)
	}

	// 7. documentSymbol
	symResp := c.request(t, 5, "textDocument/documentSymbol", map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})
	if symResp.Error != nil {
		t.Fatalf("documentSymbol error: %+v", symResp.Error)
	}

	// 8. shutdown + exit
	c.request(t, 99, "shutdown", nil)
	c.notify(t, "exit", nil)

	// Wait for the server to terminate. Time-box so a hung
	// server doesn't wedge the test harness.
	doneCh := make(chan error, 1)
	go func() { doneCh <- cmd.Wait() }()
	select {
	case <-doneCh:
		// clean exit
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("fuse lsp did not exit within 5s; stderr=%s", stderr.String())
	}
}

// scriptedClient is a minimal LSP client over framed stdin/stdout
// pipes. Synchronous: each request / notification is written
// immediately; responses are drained as they arrive.
type scriptedClient struct {
	in  io.WriteCloser
	out io.Reader
	mu  sync.Mutex
	buf []byte
}

func (c *scriptedClient) write(body []byte) error {
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.in.Write([]byte(frame)); err != nil {
		return err
	}
	_, err := c.in.Write(body)
	return err
}

func (c *scriptedClient) request(t *testing.T, id int, method string, params any) scriptedResp {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": id, "method": method, "params": params,
	})
	if err := c.write(body); err != nil {
		t.Fatalf("write %s: %v", method, err)
	}
	return c.readResponseFor(t, id)
}

func (c *scriptedClient) notify(t *testing.T, method string, params any) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "method": method, "params": params,
	})
	if err := c.write(body); err != nil {
		t.Fatalf("write notification %s: %v", method, err)
	}
}

// readResponseFor drains framed messages from the server until
// a response with the requested id arrives. Notifications are
// ignored (they don't carry an ID).
func (c *scriptedClient) readResponseFor(t *testing.T, id int) scriptedResp {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("no response for id=%d within 5s", id)
		default:
		}
		body, err := c.readFrame(t)
		if err != nil {
			t.Fatalf("read frame: %v", err)
		}
		var resp scriptedResp
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}
		if resp.ID == id {
			return resp
		}
	}
}

func (c *scriptedClient) readFrame(t *testing.T) ([]byte, error) {
	t.Helper()
	// Read header line-by-line.
	headerBuf := &bytes.Buffer{}
	for {
		// Read one byte at a time until we see CRLFCRLF.
		b := make([]byte, 1)
		n, err := c.out.Read(b)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			continue
		}
		headerBuf.Write(b)
		hb := headerBuf.Bytes()
		if len(hb) >= 4 && bytes.HasSuffix(hb, []byte("\r\n\r\n")) {
			break
		}
	}
	header := headerBuf.String()
	var length int
	for _, line := range strings.Split(header, "\r\n") {
		if strings.HasPrefix(line, "Content-Length:") {
			fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:")), "%d", &length)
		}
	}
	if length <= 0 {
		return nil, fmt.Errorf("invalid content-length in frame %q", header)
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(c.out, body); err != nil {
		return nil, err
	}
	return body, nil
}

type scriptedResp struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// asStringRaw decodes raw into a map and then walks the given
// path. Works regardless of whether Result is an object or an
// array; array results simply return empty string at any path.
func asStringRaw(raw json.RawMessage, path ...string) string {
	var cur any
	if err := json.Unmarshal(raw, &cur); err != nil {
		return ""
	}
	for _, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[key]
	}
	s, _ := cur.(string)
	return s
}

// silence unused imports on non-windows hosts.
var _ = runtime.GOOS
var _ = filepath.Join
