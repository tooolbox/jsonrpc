package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

type Echoer struct{}

func (Echoer) Echo(s string) string {
	return s
}

func TestJSONRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), "data", "Hello world!")

	// Test some sample methods.
	h := NewHandler(&Echoer{})

	h.RegisterMethod("echo", func(s string) string {
		return s
	})
	h.RegisterMethod("multiecho", func(s ...string) string {
		return strings.Join(s, " ")
	})
	h.RegisterMethod("ctx.data", func(ctx context.Context) (string, error) {
		return ctx.Value("data").(string), nil
	})
	h.RegisterMethod("nil.error", func() (string, error) {
		var err *Error
		return "Hello world!", err
	})
	h.RegisterMethod("nil.result", func() {})

	// Prepare test cases.
	type compare struct {
		In  string
		Out string
	}
	for i, c := range []compare{
		{`{
			"jsonrpc": "2.0",
			"method": "echo",
			"params": "Hello world!"
		}`, `{
			"jsonrpc": "2.0",
			"id": null,
			"result": "Hello world!"
		}`},
		{`{
			"jsonrpc": "2.0",
			"method": "Echoer.Echo",
			"params": "Hello world!"
		}`, `{
			"jsonrpc": "2.0",
			"id": null,
			"result": "Hello world!"
		}`},
		{`{
			"jsonrpc": "2.0",
			"id": "1",
			"method": "multiecho",
			"params": ["Hello", "world!"]
		}`, `{
			"jsonrpc": "2.0",
			"id": "1",
			"result": "Hello world!"
		}`},
		{`{
			"jsonrpc": "2.0",
			"id": 2,
			"method": "ctx.data"
		}`, `{
			"jsonrpc": "2.0",
			"id": 2,
			"result": "Hello world!"
		}`},
		{`{
			"jsonrpc": "2.0",
			"method": "nil.result"
		}`, `{
			"jsonrpc": "2.0",
			"id": null,
			"result": null
		}`},
	} {
		req := httptest.NewRequest("POST", "/", strings.NewReader(c.In))
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		t.Logf("Running test %d", i)
		expectJSON(t, w.Body, c.Out)
	}

	(func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("nil *jsonrpc.Error did not panic")
			}
		}()
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{
			"jsonrpc": "2.0",
			"id": 3,
			"method": "nil.error"
		}`))
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	})()
}

func expectJSON(t *testing.T, in *bytes.Buffer, expected string) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(expected)); err != nil {
		t.Fatal(err, expected)
	}
	want := buf.String()

	buf.Reset()
	if err := json.Compact(&buf, in.Bytes()); err != nil {
		t.Fatal(err, in.String())
	}
	got := buf.String()

	if got != want {
		t.Fatalf("expected: %s\ngot: %s", want, got)
	}
}
