package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/aiocloud/stream/app/api"
	"github.com/aiocloud/stream/app/log"

)

var (
	mux       *dns.ServeMux
	tcpSocket *dns.Server
	udpSocket *dns.Server

	dialer = net.Dialer{
		Timeout:   time.Second * 10,
		KeepAlive: time.Second * 10,

		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return net.Dial("udp", api.StreamData.DNS.Upstream)
			},
		},
	}
)

func Run() {
	if api.StreamData.DNS.Listen == "" {
		return
	}

	mux = dns.NewServeMux()
	mux.HandleFunc("in-addr.arpa.", handleServerName)
	mux.HandleFunc(".", handleDomainName)

	tcpSocket = &dns.Server{Net: "tcp", Addr: api.StreamData.DNS.Listen, Handler: mux}
	udpSocket = &dns.Server{Net: "udp", Addr: api.StreamData.DNS.Listen, Handler: mux}

	go func() { log.Fatal("[Stream][DNS][TCP] Error:", tcpSocket.ListenAndServe()) }()
	go func() { log.Fatal("[Stream][DNS][UDP] Error:", udpSocket.ListenAndServe()) }()

	log.Info("[Stream][DNS] Started")
}

func Dial(network, address string) (net.Conn, error) {
	return dialer.Dial(network, address)
}

func handleServerName(w dns.ResponseWriter, r *dns.Msg) {
	if api.StreamData.DNS.Strict {
		checked, err := api.CheckIP(w.RemoteAddr())
		if err != nil {
			log.Info("[Stream][DNS][api.CheckIP] Error:", err)
			return
		}

		if !checked {
			return
		}
	}

	m := new(dns.Msg)
	m.SetReply(r)

	for i := 0; i < len(r.Question); i++ {
		rr, err := dns.NewRR(fmt.Sprintf("%s PTR aioCloud", r.Question[i].Name))
		if err != nil {
			log.Info(err)
			return
		}

		m.Answer = append(m.Answer, rr)
	}

	_ = w.WriteMsg(m)
}

func handleDomainName(w dns.ResponseWriter, r *dns.Msg) {
	if api.StreamData.DNS.Strict {
		checked, err := api.CheckIP(w.RemoteAddr())
		if err != nil {
			log.Info("[Stream][DNS][handleDomainName] api.CheckIP:", err)
			return
		}

		if !checked {
			return
		}
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Question = make([]dns.Question, 0)

	for i := 0; i < len(r.Question); i++ {
		if r.Question[i].Qtype == dns.TypeA || r.Question[i].Qtype == dns.TypeAAAA {
			if checked, _ := api.CheckDomain(strings.TrimRight(r.Question[i].Name, "."), "0"); checked {
				var rr dns.RR

				if r.Question[i].Qtype == dns.TypeA {
					if api.CurrentIPv4 != "" {
						rr, _ = dns.NewRR(fmt.Sprintf("%s A %s", r.Question[i].Name, api.CurrentIPv4))
					} else {
						rr, _ = dns.NewRR(fmt.Sprintf("%s A 0.0.0.0", r.Question[i].Name))
					}
				}

				if r.Question[i].Qtype == dns.TypeAAAA {
					if api.CurrentIPv6 != "" {
						rr, _ = dns.NewRR(fmt.Sprintf("%s AAAA %s", r.Question[i].Name, api.CurrentIPv6))
					} else {
						rr, _ = dns.NewRR(fmt.Sprintf("%s AAAA ::", r.Question[i].Name))
					}
				}

				m.Question = append(m.Question, r.Question[i])
				m.Answer = append(m.Answer, rr)
			}
		}
	}

	if len(m.Answer) > 0 {
		w.WriteMsg(m)
		return
	}

	m, err := dns.Exchange(r, api.StreamData.DNS.Upstream)
	if err != nil {
		return
	}

	w.WriteMsg(m)
}
