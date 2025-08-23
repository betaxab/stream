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
	"github.com/aiocloud/stream/app"
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
	client.SetHeader("User-Agent", fmt.Sprintf("Stream/%s", app.Version))

	resp, err := client.R().Get(url)
	if err != nil {
		return nil, fmt.Errorf("http.Get: %w", err)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("http.Get: unexpected status %d", resp.StatusCode())
	}

	// Cloudflare trace 是纯文本 key=value 按行返回
	scanner := bufio.NewScanner(bytes.NewReader(resp.Body()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(k), "ip") {
			ip := net.ParseIP(strings.TrimSpace(v))
			if ip == nil {
				return nil, fmt.Errorf("http.Get: parse ip failed: %q", v)
			}
			return ip, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("http.Get: read body: %w", err)
	}
	return nil, fmt.Errorf("http.Get: ip field not found in response")
}

// GetInterfaceIP 获取指定网卡的 IPv4 地址
func GetInterfaceIP(interfaceName string, ipType string) (net.IP, error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("api.GetInterfaceIP: Get IP address failed: %w", err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("api.GetInterfaceIP: Get IP address failed: %w", err)
	}

	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			ip := v.IP
			if strings.ToLower(ipType) == "ipv4" && ip.To4() != nil {
				return ip, nil
			}
			if strings.ToLower(ipType) == "ipv6" && ip.To16() != nil && ip.To4() == nil {
				return ip, nil
			}
		case *net.IPAddr:
			ip := v.IP
			if strings.ToLower(ipType) == "ipv4" && ip.To4() != nil {
				return ip, nil
			}
			if strings.ToLower(ipType) == "ipv6" && ip.To16() != nil && ip.To4() == nil {
				return ip, nil
			}
		}
	}

	return nil, fmt.Errorf("api.GetInterfaceIP: Interface %s does not support IP Type: %s", interfaceName, ipType)
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
	var addr net.IP
	var err error

	if StreamData.DNS.UseInterfaceIP && StreamData.DNS.InterfaceName != "" {
		addr, err = GetInterfaceIP(StreamData.DNS.InterfaceName, "ipv4")
	} else {
		addr, err = GetIP(StreamData.API.IPv4)
	}

	if err != nil {
		return fmt.Errorf("api.GetIP: %w", err)
	}

	CurrentIPv4 = addr.String()
	return nil
}

func UpdateIPv6() error {
	var addr net.IP
	var err error

	if StreamData.DNS.UseInterfaceIP && StreamData.DNS.InterfaceName != "" {
		addr, err = GetInterfaceIP(StreamData.DNS.InterfaceName, "ipv6")
	} else {
		addr, err = GetIP(StreamData.API.IPv6)
	}

	if err != nil {
		return fmt.Errorf("api.GetIP: %w", err)
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
