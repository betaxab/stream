package mitm

import (
	"bytes"
	"io"
	"net"
	"strings"
	"time"

	"github.com/aiocloud/stream/api"
	"github.com/aiocloud/stream/dns"
	"github.com/aiocloud/stream/log"
)

func handleHTTP(client net.Conn) {
	defer client.Close()

	data := make([]byte, 1446)
	size, err := client.Read(data)
	if err != nil {
		return
	}
	data = data[:size]

	offset := bytes.Index(data, []byte{0x0d, 0x0a, 0x0d, 0x0a})
	if offset == -1 {
		return
	}

	list := make(map[string]string)
	{
		hdr := bytes.Split(data[:offset], []byte{0x0d, 0x0a})
		for i := 0; i < len(hdr); i++ {
			if i == 0 {
				continue
			}

			SPL := strings.SplitN(string(hdr[i]), ":", 2)
			if len(SPL) < 2 {
				continue
			}

			list[strings.ToUpper(strings.TrimSpace(SPL[0]))] = strings.TrimSpace(SPL[1])
		}
	}

	if _, ok := list["HOST"]; !ok {
		return
	}
	host := list["HOST"]

	_, s, _ := net.SplitHostPort(client.LocalAddr().String())
	checked, outbound := api.CheckDomain(host, s)
	if api.StreamData.Strict {
		if !checked {
			return
		}
	} else {
		outbound = "DIRECT"
	}

	var remote net.Conn
	if outbound == "DIRECT" {
		remote, err = dns.Dial(api.StreamData.Out.Network, net.JoinHostPort(host, s))
		if err != nil {
			return
		}
	} else {
		remote, err = dns.Dial(api.StreamData.Out.Network, outbound)
		if err != nil {
			return
		}
	}
	defer remote.Close()

	if _, err := remote.Write(data); err != nil {
		return
	}
	data = nil

	log.Infof("[Stream][HTTP][%s] %s <-> %s (%s)", s, client.RemoteAddr(), remote.RemoteAddr(), host)

	go func() {
		io.CopyBuffer(client, remote, make([]byte, 1446))
		client.SetDeadline(time.Now())
		remote.SetDeadline(time.Now())
	}()

	io.CopyBuffer(remote, client, make([]byte, 1446))
	client.SetDeadline(time.Now())
	remote.SetDeadline(time.Now())
}
