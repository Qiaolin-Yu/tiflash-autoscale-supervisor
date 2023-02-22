package main

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func mockMetricHttpServer(inputFile string) (*httptest.Server, error) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, err
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, err := rw.Write(data)
		if err != nil {
			return
		}
	}))
	return server, nil
}

func TestGetTiFlashTaskNum(t *testing.T) {
	server, err := mockMetricHttpServer("test_data/metrics1.txt")
	assert.NoError(t, err)

	TiFlashMetricURL = server.URL

	res, err := GetTiFlashTaskNum()
	assert.NoError(t, err)
	assert.Equal(t, res, 0)
	server.Close()

	server, err = mockMetricHttpServer("test_data/metrics2.txt")
	assert.NoError(t, err)
	TiFlashMetricURL = server.URL
	res, err = GetTiFlashTaskNum()
	assert.NoError(t, err)
	assert.Equal(t, res, 66)
	server.Close()
}
