package main

import (
	"os"
	"testing"
)

func TestGetTiFlashTaskNumByMetricsByte(t *testing.T) {
	data1, err := os.ReadFile("test_data/metrics1.txt")
	if err != nil {
		t.Error(err)
	}
	res, err := GetTiFlashTaskNumByMetricsByte(data1)
	if err != nil {
		t.Error(err)
	}
	if res != 0 {
		t.Error("TiFlash task num should be 0")
	}
	data2, err := os.ReadFile("test_data/metrics2.txt")
	if err != nil {
		t.Error(err)
	}
	res, err = GetTiFlashTaskNumByMetricsByte(data2)
	if err != nil {
		t.Error(err)
	}
	if res != 66 {
		t.Error("TiFlash task num should be 66")
	}
}
