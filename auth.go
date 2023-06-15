package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Auth(path string, public ...ed25519.PublicKey) (filePath string, err error) {
	if len(public) == 0 {
		return path, nil
	}
	path = strings.Trim(path, "/")
	for _, pk := range public {
		switch err = auth(filepath.Dir(path), pk); err {
		case errNoAuth:
			continue
		case nil:
			return "/" + strings.SplitN(path, "/", 3)[2], nil
		default:
			break
		}
	}
	return "", err
}
func genAuth(dir string, deadline time.Time, key ed25519.PrivateKey) string {
	var token string
	if strings.HasPrefix(dir, "/") {
		token = strconv.FormatInt(deadline.UTC().Unix(), 10) + dir
	} else {
		token = strconv.FormatInt(deadline.UTC().Unix(), 10) + "/" + dir
	}
	sig := base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(token)))
	return sig + "/" + token
}

var errNoAuth = errors.New("no auth")

func auth(dir string, public ed25519.PublicKey) error {
	parts := strings.SplitN(dir, "/", 2)
	if len(parts) != 2 {
		return errNoAuth
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}
	token := parts[1]
	parts = strings.SplitN(parts[1], "/", 2)
	if len(parts) != 2 {
		return errors.New("no timestamp")
	}
	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return err
	}
	if timestamp < 0 || timestamp < time.Now().UTC().Unix() {
		return errors.New("past timestamp")
	}
	if !ed25519.Verify(public, []byte(token), sig) {
		return errors.New("unauthorized")
	}
	return nil
}
