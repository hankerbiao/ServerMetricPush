package pusher

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewPusher(t *testing.T) {
	pusher := NewPusher("http://localhost:9091")
	if pusher == nil {
		t.Error("expected non-nil pusher")
	}
	if pusher.url != "http://localhost:9091" {
		t.Errorf("expected url http://localhost:9091, got %s", pusher.url)
	}
	if pusher.job != "node" {
		t.Errorf("expected default job 'node', got %s", pusher.job)
	}
}

func TestNewPusher_WithJob(t *testing.T) {
	pusher := NewPusher("http://localhost:9091", WithJob("custom-job"))
	if pusher.job != "custom-job" {
		t.Errorf("expected job 'custom-job', got %s", pusher.job)
	}
}

func TestNewPusher_WithInstance(t *testing.T) {
	pusher := NewPusher("http://localhost:9091", WithInstance("server1"))
	if pusher.instance != "server1" {
		t.Errorf("expected instance 'server1', got %s", pusher.instance)
	}
}

func TestNewPusher_WithTimeout(t *testing.T) {
	pusher := NewPusher("http://localhost:9091", WithTimeout(5*time.Second))
	if pusher.httpClient.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5, got %v", pusher.httpClient.Timeout)
	}
}

func TestPusher_Push_Success(t *testing.T) {
	pusher := NewPusher("http://pushgateway:9091")
	pusher.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.String() != "http://pushgateway:9091/metrics/job/node" {
			t.Errorf("unexpected url %s", r.URL.String())
		}
		if r.Header.Get("Content-Type") != "text/plain; version=0.0.4" {
			t.Errorf("expected Content-Type text/plain; version=0.0.4, got %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Equal(body, []byte("metric 123")) {
			t.Errorf("unexpected body %q", string(body))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBuffer(nil)),
			Header:     make(http.Header),
		}, nil
	})}
	err := pusher.Push([]byte("metric 123"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPusher_Push_Failure(t *testing.T) {
	pusher := NewPusher("http://pushgateway:9091")
	pusher.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString("bad request")),
			Header:     make(http.Header),
		}, nil
	})}
	err := pusher.Push([]byte("metric 123"))
	if err == nil {
		t.Error("expected error for bad request")
	}
}

func TestPusher_Push_InvalidURL(t *testing.T) {
	pusher := NewPusher("://invalid")
	err := pusher.Push([]byte("metric 123"))
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestPusher_Push_WithInstance(t *testing.T) {
	pusher := NewPusher("http://pushgateway:9091", WithInstance("server1"))
	pusher.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/metrics/job/node/instance/server1" {
			t.Errorf("expected path /metrics/job/node/instance/server1, got %s", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBuffer(nil)),
			Header:     make(http.Header),
		}, nil
	})}

	err := pusher.Push([]byte("metric 123"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPusher_Push_ContentType(t *testing.T) {
	pusher := NewPusher("http://pushgateway:9091")
	pusher.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "text/plain; version=0.0.4" {
			t.Errorf("expected Content-Type 'text/plain; version=0.0.4', got '%s'", contentType)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBuffer(nil)),
			Header:     make(http.Header),
		}, nil
	})}
	err := pusher.Push([]byte("test"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
