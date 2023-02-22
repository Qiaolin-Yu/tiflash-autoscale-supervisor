package main

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetTiFlashTaskNumHttp(t *testing.T) {
	data, err := os.ReadFile("test_data/metrics2.txt")
	assert.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, err := rw.Write(data)
		if err != nil {
			return
		}
	}))
	TiFlashMetricURL = server.URL

	defer server.Close()
	res, err := GetTiFlashTaskNum()
	assert.NoError(t, err)
	assert.Equal(t, res, 66)
}

func TestGetTiFlashTaskNumByMetricsByte(t *testing.T) {
	data1, err := os.ReadFile("test_data/metrics1.txt")
	assert.NoError(t, err)
	res, err := GetTiFlashTaskNumByMetricsByte(data1)
	assert.NoError(t, err)
	assert.Equal(t, res, 0)
	data2, err := os.ReadFile("test_data/metrics2.txt")
	assert.NoError(t, err)
	res, err = GetTiFlashTaskNumByMetricsByte(data2)
	assert.NoError(t, err)
	assert.Equal(t, res, 66)
}
