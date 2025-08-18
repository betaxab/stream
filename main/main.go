package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/aiocloud/stream/app"
	"github.com/aiocloud/stream/app/api"
	"github.com/aiocloud/stream/app/conf"
	"github.com/aiocloud/stream/app/log"
	"github.com/aiocloud/stream/app/dns"
	"github.com/aiocloud/stream/app/mitm"
)

type Initializer func() error

var initializers = []Initializer{conf.Init, log.Init}

func init() {
	for _, initFunc := range initializers {
		if err := initFunc(); err != nil {

			panic(fmt.Sprintf("[Stream] Initial failed %v", err))
		}
	}
}

func main() {

	api.Run()
	dns.Run()

	for _, i := range api.StreamData.TCP.TLS {
		go mitm.ListenTLS(i)
	}

	for _, i := range api.StreamData.TCP.HTTP {
		go mitm.ListenHTTP(i)
	}

	go UpdateIP()
	go UpdateRule()

	log.Infof("[Stream] IPv4: %s IPv6: %s", api.CurrentIPv4, api.CurrentIPv6)
	log.Info("[Stream][Main] Started, Version:", app.Version)
	fmt.Println("[Stream][Main] Started, Version: " + app.Version)

	for {
		time.Sleep(time.Minute * 10)

		runtime.GC()

		stats := new(runtime.MemStats)
		runtime.ReadMemStats(stats)
		log.Infof("[Stream][GC] CPU Fraction %f", stats.GCCPUFraction)
		log.Infof("[Stream][GC] Obtained %dMB", stats.Sys/1024/1024)
		log.Infof("[Stream][GC] Assigned %dMB", stats.Alloc/1024/1024)
		log.Infof("[Stream][GC] Routine %d", runtime.NumGoroutine())
	}
}

func UpdateIP() {
	for {
		time.Sleep(time.Second * 120)

		if err := api.UpdateIPv4(); err != nil {
			log.Info("[Stream] api.UpdateIPv4:", err)
		}

		if err := api.UpdateIPv6(); err != nil {
			log.Info("[Stream] api.UpdateIPv6:", err)
		}

		log.Infof("[Stream] IPv4: %s IPv6: %s", api.CurrentIPv4, api.CurrentIPv6)
	}
}

func UpdateRule() {
	for {
		time.Sleep(time.Second * 86400)

		if err := api.UpdateRule(); err != nil {
			log.Info("[Stream] Update rule failed:", err)
		}
	}
}
