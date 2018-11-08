package main

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

func main() {
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:11080",
		// &proxy.Auth{User: "", Password: ""},
		nil,
		&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	)
	if err != nil {
		log.Println(err)
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
		log.Println(err)
		return
	}
	content, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("My public IP is", "\""+strings.TrimSpace(string(content))+"\"")
	}
}
