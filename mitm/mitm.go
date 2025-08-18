package mitm

import (
	"net"

	"github.com/aiocloud/stream/api"
	"github.com/aiocloud/stream/log"
)

func ListenHTTP(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("[Stream][HTTP][net.Listen] Error:", err)
	}
	defer ln.Close()

	log.Infof("[Stream][HTTP][%s] Started", addr)

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Fatal("[Stream][HTTP][ln.Accept] Error:", err)
		}

		if checked, err := api.CheckIP(client.RemoteAddr()); err != nil {
			log.Info("[Stream][HTTP][api.CheckIP]:", err)

			_ = client.Close()
			continue
		} else if !checked {
			log.Info("[Stream][HTTP][api.CheckIP] Ban:", client.RemoteAddr())

			_ = client.Close()
			continue
		}

		go handleHTTP(client)
	}
}

func ListenTLS(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("[Stream][TLS][net.Listen] Error:", err)
	}
	defer ln.Close()

	log.Infof("[Stream][TLS][%s] Started", addr)

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Fatal("[Stream][TLS][ln.Accept] Error:", err)
		}

		if checked, err := api.CheckIP(client.RemoteAddr()); err != nil {
			log.Info("[Stream][TLS][api.CheckIP]:", err)

			_ = client.Close()
			continue
		} else if !checked {
			log.Info("[Stream][TLS][api.CheckIP] Ban:", client.RemoteAddr())

			_ = client.Close()
			continue
		}

		go handleTLS(client)
	}
}
