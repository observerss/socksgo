package main

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/proxy"
)

func Test_Sys(t *testing.T) {
	// prepare server
	listenAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:11080")
	bindAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	app := &SocksProxy{
		ListenAddr:  listenAddr,
		BindAddr:    bindAddr,
		Username:    "",
		Password:    "",
		AllowNoAuth: true,
	}
	go func() {
		err := app.RunServer()
		if err != nil {
			t.Error(err)
		}
	}()

	time.Sleep(time.Duration(1) * time.Millisecond)

	// run client
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:11080",
		// &proxy.Auth{User: "", Password: ""},
		nil,
		&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	)
	if err != nil {
		t.Error(err)
		return
	}
	transport := &http.Transport{
		Proxy:               nil,
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	client := &http.Client{Transport: transport}
	resp, err := client.Get("http://members.3322.org/dyndns/getip")
	if err != nil {
		t.Error(err)
		return
	}
	content, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Error(err)
	} else {
		log.Println("My public IP is", "\""+strings.TrimSpace(string(content))+"\"")
	}

	// stop server
	// app.StopServer()
}
