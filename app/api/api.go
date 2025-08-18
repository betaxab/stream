package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/yl2chen/cidranger"
)

var (
	CurrentIPv4 string
	CurrentIPv6 string

	StreamData Stream

	cidr  cidranger.Ranger
	mutex sync.RWMutex
)

func Run() {
	if StreamData.API.Listen == "" {
		return
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Any("/addr", func(c *gin.Context) {
		c.String(http.StatusOK, "ipv4=%s\nipv6=%s\n", CurrentIPv4, CurrentIPv6)
	})
	r.Any("/aio", func(c *gin.Context) {
		var addr net.IP = nil

		if data := c.Request.Header.Get("X-Real-IP"); data != "" {
			addr = net.ParseIP(data)
		} else {
			host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
			if err != nil {
				return
			}

			addr = net.ParseIP(host)
		}

		mutex.RLock()
		if checked, _ := cidr.Contains(addr); checked {
			mutex.RUnlock()

			c.String(http.StatusOK, "DONE %s\n", addr.String())
			return
		}

		data := c.Query("secret")
		for _, i := range StreamData.API.Secret {
			if data == i {
				mutex.RUnlock()

				asn := ""
				if addr.To4() != nil {
					asn = addr.String() + "/32"
				} else {
					asn = addr.String() + "/128"
				}

				_, ipn, err := net.ParseCIDR(asn)
				if err != nil {
					c.String(http.StatusBadRequest, "net.ParseCIDR: %v", err)
				}

				mutex.Lock()
				cidr.Insert(cidranger.NewBasicRangerEntry(*ipn))
				mutex.Unlock()

				c.String(http.StatusOK, "DONE %s\n", addr.String())
				return
			}
		}

		mutex.RUnlock()
		c.String(http.StatusForbidden, "FAIL\n")
	})
	r.Any("/", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	go func() {
		log.Fatalf("[Stream][API][Run][r.Run] %v", r.Run(StreamData.API.Listen))
	}()

	log.Println("[Stream][API] Started")
}

func Load(name string) error {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return fmt.Errorf("ioutil.ReadFile: %v", err)
	}

	if err = json.Unmarshal(data, &StreamData); err != nil {
		return fmt.Errorf("json.Unmarshal: %v", err)
	}

	mutex.Lock()
	defer mutex.Unlock()

	cidr = cidranger.NewPCTrieRanger()
	for _, i := range StreamData.List {
		_, ipn, err := net.ParseCIDR(i)
		if err != nil {
			return fmt.Errorf("net.ParseCIDR: %v", err)
		}

		cidr.Insert(cidranger.NewBasicRangerEntry(*ipn))
	}

	return nil
}

func GetIP(url string) (net.IP, error) {
	client := resty.New()
	client.SetTimeout(time.Second * 10)

	response, err := client.R().Get(url)
	if err != nil {
		return nil, fmt.Errorf("http.Get: %v", err)
	}

	var addr net.IP = nil
	scanner := bufio.NewReader(bytes.NewReader(response.Body()))
	for {
		i, _, _ := scanner.ReadLine()
		if i == nil {
			break
		}

		list := strings.SplitN(string(i), "=", 2)
		if list[0] == "ip" {
			addr = net.ParseIP(strings.TrimSpace(list[1]))
			break
		}
	}

	return addr, nil
}

func UpdateRule() error {
	for i := 0; i < len(StreamData.Rule); i++ {
		if err := StreamData.Rule[i].Update(); err != nil {
			return fmt.Errorf("r.Update: %v", err)
		}
	}

	return nil
}

func UpdateIPv4() error {
	addr, err := GetIP(StreamData.API.IPv4)
	if err != nil {
		return fmt.Errorf("api.GetIP: %v", err)
	}

	CurrentIPv4 = addr.String()
	return nil
}

func UpdateIPv6() error {
	addr, err := GetIP(StreamData.API.IPv6)
	if err != nil {
		return fmt.Errorf("api.GetIP: %v", err)
	}

	CurrentIPv6 = addr.String()
	return nil
}

func CheckIP(name net.Addr) (bool, error) {
	addr, _, err := net.SplitHostPort(name.String())
	if err != nil {
		return false, fmt.Errorf("net.SplitHostPort: %v", err)
	}

	mutex.RLock()
	defer mutex.RUnlock()

	checked, err := cidr.Contains(net.ParseIP(addr))
	if err != nil {
		return false, fmt.Errorf("cidr.Contains: %v", err)
	}

	return checked, nil
}

func CheckDomain(host string, port string) (bool, string) {
	mutex.RLock()
	defer mutex.RUnlock()

	for i := 0; i < len(StreamData.Rule); i++ {
		checked, outbound := StreamData.Rule[i].Search(host, port)
		if checked {
			return true, outbound
		}
	}

	return false, ""
}
