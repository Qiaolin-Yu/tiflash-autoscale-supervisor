package main

import (
	"log"
	"testing"
)

func TestGetTiFlashTaskNum(t *testing.T) {
	num, err := GetTiFlashTaskNum()
	if err != nil {
		log.Fatalf("get tiflash task num failed: %v", err)
	}
	log.Printf("tiflash task num: %d", num)
}
