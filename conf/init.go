package conf

import (
	"flag"
	"fmt"

	"github.com/aiocloud/stream/api"
)

var (
	Path string
)

const (
	defaultOutputLog = "/var/log/stream.log"
)

func Init() error {
	flag.StringVar(&Path, "c", "/etc/stream.json", "Path")
	flag.Parse()

	if err := api.Load(Path); err != nil {
		return fmt.Errorf("[Stream] api.Load Error:", err)
	}

	if err := api.UpdateIPv4(); err != nil {
		fmt.Println("[Stream] api.UpdateIPv4 Error:", err)
	}

	if err := api.UpdateIPv6(); err != nil {
		fmt.Println("[Stream] api.UpdateIPv6 Error:", err)
	}

	if api.CurrentIPv4 == "" && api.CurrentIPv6 == "" {
		return fmt.Errorf("[Stream] Get current ip address failed")
	}

	if err := api.UpdateRule(); err != nil {
		return fmt.Errorf("[Stream] Update rule failed:", err)
	}

	return nil
}

func GetOutputLog() string {
	outputlog := api.StreamData.Log.OutputLog
	if outputlog == "" {
		outputlog = defaultOutputLog
	}

	return outputlog
}

