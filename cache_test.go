package main

import (
	"testing"
)

func TestCache(t *testing.T) {
	err := InitCache(MaimaiSource("tests"))
	if err != nil {
		t.Error(err)
	}
}
