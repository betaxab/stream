package mitm

import (
	"io"
	"net"
	"time"

	"github.com/aiocloud/stream/api"
	"github.com/aiocloud/stream/log"
	"github.com/aiocloud/stream/dns"
)

func handleTLS(client net.Conn) {
	defer client.Close()

	data := make([]byte, 1446)
	size, err := client.Read(data)
	if err != nil || size <= 44 {
		return
	}
	data = data[:size]

	if data[0] != 0x16 {
		return
	}

	offset := 0
	offset += 1 // Content Type
	offset += 2 // Version
	offset += 2 // Length

	// Handshake Type
	if data[offset] != 0x01 {
		return
	}
	offset += 1

	offset += 3  // Length
	offset += 2  // Version
	offset += 32 // Random

	// Session ID
	length := int(data[offset])
	offset += 1
	offset += length
	if size <= offset+1 {
		return
	}

	// Cipher Suites
	length = (int(data[offset]) << 8) + int(data[offset+1])
	offset += 2
	offset += length
	if size <= offset {
		return
	}

	// Compression Methods
	length = int(data[offset])
	offset += 1
	offset += length

	// Extension Length
	offset += 2
	if size <= offset+1 {
		return
	}

	host := ""
	for size > offset+2 && host == "" {
		// Extension Type
		name := (int(data[offset]) << 8) + int(data[offset+1])
		offset += 2
		if size <= offset+1 {
			return
		}

		// Extension Length
		length = (int(data[offset]) << 8) + int(data[offset+1])
		offset += 2

		// Extension: Server Name
		if name == 0 {
			// Server Name List Length
			offset += 2
			if size <= offset {
				return
			}

			// Server Name Type
			if data[offset] != 0x00 {
				return
			}
			offset += 1
			if size <= offset+1 {
				return
			}

			// Server Name Length
			length = (int(data[offset]) << 8) + int(data[offset+1])
			offset += 2
			if size <= offset+length {
				return
			}

			// Server Name
			host = string(data[offset : offset+length])

			// Get Out
			break
		}

		// Extension Data
		offset += length
	}

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

	log.Infof("[Stream][TLS][%s] %s <-> %s (%s)", s, client.RemoteAddr().String(), remote.RemoteAddr().String(), host)

	go func() {
		io.CopyBuffer(client, remote, make([]byte, 1446))
		client.SetDeadline(time.Now())
		remote.SetDeadline(time.Now())
	}()

	io.CopyBuffer(remote, client, make([]byte, 1446))
	client.SetDeadline(time.Now())
	remote.SetDeadline(time.Now())
}
