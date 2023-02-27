package main

import (
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var (
	httpServerURL = "127.0.0.1:8234"
)

func NewTestServerWithURL(url string, handler http.Handler) (*httptest.Server, error) {
	ts := httptest.NewUnstartedServer(handler)
	if url != "" {
		l, err := net.Listen("tcp", url)
		if err != nil {
			return nil, err
		}
		ts.Listener.Close()
		ts.Listener = l
	}
	ts.Start()
	return ts, nil
}

func mockMetricHttpServer(inputFile string) (*httptest.Server, error) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, err
	}
	server, err := NewTestServerWithURL(httpServerURL, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, err := rw.Write(data)
		if err != nil {
			return
		}
	}))
	return server, err
}

func TestGetTiFlashTaskNum(t *testing.T) {
	server, err := mockMetricHttpServer("test_data/metrics1.txt")
	assert.NoError(t, err)
	res, err := GetTiFlashTaskNum()
	assert.NoError(t, err)
	assert.Equal(t, res, 0)
	server.Close()

	server, err = mockMetricHttpServer("test_data/metrics2.txt")
	assert.NoError(t, err)
	res, err = GetTiFlashTaskNum()
	assert.NoError(t, err)
	assert.Equal(t, res, 66)
	server.Close()
}
