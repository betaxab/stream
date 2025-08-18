package api

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type Stream struct {
	API    StreamAPI    `json:"api"`
	Log    StreamLog     `json:"log"`
	DNS    StreamDNS    `json:"dns"`
	TCP    StreamTCP    `json:"tcp"`
	Out    StreamOut    `json:"out"`
	Rule   []StreamRule `json:"rules"`
	List   []string     `json:"whitelist"`
	Strict bool         `json:"strict"`
}

type StreamAPI struct {
	Listen string   `json:"listen"`
	Secret []string `json:"secret"`
	IPv4   string   `json:"ipv4"`
	IPv6   string   `json:"ipv6"`
}

type StreamLog struct {
	OutputLog string `json:"output"`
}

type StreamDNS struct {
	Strict   bool   `json:"strict"`
	Listen   string `json:"listen"`
	Upstream string `json:"upstream"`
}

type StreamTCP struct {
	TLS  []string `json:"tls"`
	HTTP []string `json:"http"`
}

type StreamOut struct {
	Network string `json:"network"`
}

type StreamRule struct {
	URL     string     `json:"url"`
	Mapping []PortRule `json:"mapping"`
	List    []Rule     `json:"-"`
}

func (r *StreamRule) Search(host string, port string) (bool, string) {
	for i := 0; i < len(r.List); i++ {
		if r.List[i].Search(host) {
			return true, r.Outbound(port)
		}
	}

	return false, ""
}

func (r *StreamRule) Update() error {
	client := resty.New()
	client.SetTimeout(time.Second * 10)

	response, err := client.R().Get(r.URL)
	if err != nil {
		return fmt.Errorf("http.Get: %v", err)
	}

	mutex.Lock()
	defer mutex.Unlock()

	r.List = make([]Rule, 0)
	scanner := bufio.NewReader(bytes.NewReader(response.Body()))
	for {
		i, _, _ := scanner.ReadLine()
		if i == nil {
			break
		}

		data := strings.TrimSpace(string(i))
		if len(data) == 0 {
			continue
		}

		list := strings.Split(data, ",")
		if len(list) < 2 {
			continue
		}

		switch list[0] {
		case "DOMAIN":
		case "DOMAIN-SUFFIX":
		case "DOMAIN-KEYWORD":
		default:
			continue
		}

		r.List = append(r.List, Rule{Type: list[0], Host: list[1]})
	}

	return nil
}

func (r *StreamRule) Outbound(name string) string {
	n, _ := strconv.Atoi(name)

	for _, i := range r.Mapping {
		if i.Port == 0 {
			return net.JoinHostPort(i.Addr, name)
		}

		if i.Port == n {
			return i.Addr
		}
	}

	return "DIRECT"
}

type Rule struct {
	Type string `json:"type"`
	Host string `json:"host"`
}

func (r *Rule) Search(name string) bool {
	switch r.Type {
	case "DOMAIN":
		return r.Host == name
	case "DOMAIN-SUFFIX":
		return strings.HasSuffix(name, r.Host)
	case "DOMAIN-KEYWORD":
		return strings.Contains(name, r.Host)
	default:
		return false
	}
}

type PortRule struct {
	Port int    `json:"port"`
	Addr string `json:"addr"`
}
