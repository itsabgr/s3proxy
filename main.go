package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"flag"
	"github.com/jlaffaye/ftp"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"time"
)

var flagAddr = flag.String("addr", "", "* listening address")
var flagKey = flag.String("key", "", "https key file path")
var flagCrt = flag.String("crt", "", "https certificate file path")
var flagHost = flag.String("host", "ftp(s)://[user]:[pass]@[host]:[port]/root", "* ftp host uri")
var flagCORS = flag.String("cors", "*", "'Access-Control-Allow-Origin' header value")
var flagCache = flag.String("cache", "no-store", "'Cache-Control' header value")
var flagPK = flag.String("pk", "", "ed25519 public key in hex")

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()
	flag.Parse()
	publicKey := ed25519.PublicKey(Must(hex.DecodeString(*flagPK)))
	if len(publicKey) != ed25519.PublicKeySize {
		panic(errors.New("invalid public key size"))
	}
	var ftpURL = Must(url.Parse(*flagHost))
	ftpConnPool := NewFTPPool(*ftpURL)
	ftpConnPool.Put(Must(ftpConnPool.Get(context.Background())))
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Robots-Tag", "noindex, nofollow")
		Close(request.Body)
		ctx, cancel := context.WithTimeout(request.Context(), time.Second*10)
		defer cancel()
		request = request.WithContext(ctx)
		writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET")
		writer.Header().Set("Access-Control-Allow-Origin", *flagCORS)
		switch request.Method {
		case http.MethodGet:
		case http.MethodOptions:
			writer.WriteHeader(http.StatusNoContent)
			return
		default:
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		filePath, err := Auth(request.URL.Path, publicKey)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		conn, err := ftpConnPool.Get(request.Context())
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		defer ftpConnPool.Put(conn)
		writer.Header().Set("Cache-Control", *flagCache)
		if fileInfo, err := conn.GetEntry(filePath); err != nil || fileInfo.Type != ftp.EntryTypeFile {
			http.NotFound(writer, request)
			return
		}
		resp, err := conn.Retr(filePath)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		defer Close(resp)
		if deadline, ok := request.Context().Deadline(); ok {
			if err = resp.SetDeadline(deadline); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if ext := filepath.Ext(filePath); ext != "" {
			if mimeType := mime.TypeByExtension(ext); mimeType != "" {
				writer.Header().Add("Content-Type", mimeType)
			}
		}
		if _, err := io.Copy(writer, resp); err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	})
	httpServer := http.Server{
		Addr:                         *flagAddr,
		Handler:                      handler,
		DisableGeneralOptionsHandler: true,
		ReadHeaderTimeout:            time.Second * 2,
		ReadTimeout:                  time.Second * 2,
		WriteTimeout:                 time.Second * 15,
		IdleTimeout:                  time.Second * 15,
		MaxHeaderBytes:               2000,
		ErrorLog:                     log.New(io.Discard, "", 0),
	}
	go func() {
		time.Sleep(time.Second)
		log.Println("ready")
	}()
	if *flagKey == "" {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := httpServer.ListenAndServeTLS(*flagCrt, *flagKey); err != nil {
			log.Fatal(err)
		}
	}

}
