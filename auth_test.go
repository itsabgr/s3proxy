package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestAuthCorrectness(t *testing.T) {
	public, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	token := genAuth("dir/sub/dir2", time.Now().Add(time.Minute), key)
	if err := auth(token, public); err != nil {
		t.Error(err)
	}
	fp, err := Auth("/"+token+"/file", public)
	if err != nil {
		t.Error(err)
	}
	if fp != "/dir/sub/dir2/file" {
		t.Fail()
	}
}

func TestAuthSubDir(t *testing.T) {
	public, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	token := genAuth("dir/sub/dir2", time.Now().Add(time.Minute), key)
	if err := auth(token, public); err != nil {
		t.Error(err)
	}
	fp, err := Auth("/"+token+"/sub/file", public)
	if err == nil {
		t.Fail()
	}
	if fp != "" {
		t.Fail()
	}
}

func TestAuthPastTime(t *testing.T) {
	public, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	token := genAuth("dir/sub/dir2", time.Now().Add(-time.Second), key)
	if err := auth(token, public); err == nil {
		t.Fail()
	}
}
