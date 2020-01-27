package main

import (
	"os"
	"fmt"
	"time"

	"github.com/aztecrabbit/liblog"
	"github.com/aztecrabbit/libutils"
	"github.com/aztecrabbit/libinject"
	"github.com/aztecrabbit/libredsocks"
	"github.com/aztecrabbit/libproxyrotator"
	"github.com/aztecrabbit/brainfuck-psiphon-pro-go/src/libpsiphon"
)

const (
	appName = "Brainfuck Tunnel"
	appVersionName = "Psiphon Pro Go"
	appVersionCode = "200127"

	copyrightYear = "2020"
	copyrightAuthor = "Aztec Rabbit"
)

var (
	InterruptHandler = new(libutils.InterruptHandler)
	Redsocks = new(libredsocks.Redsocks)
)

type Config struct {
	ProxyRotator *libproxyrotator.Config
	Inject *libinject.Config
	PsiphonCore int
	Psiphon *libpsiphon.Config
}

func init() {
	InterruptHandler.Handle = func() {
		libredsocks.Stop(Redsocks)
		libpsiphon.Stop()
		liblog.LogKeyboardInterrupt()
	}
	InterruptHandler.Start()
}

func main() {
	liblog.Header(
		[]string{
			fmt.Sprintf("%s [%s Version. %s]", appName, appVersionName, appVersionCode),
			fmt.Sprintf("(c) %s %s.", copyrightYear, copyrightAuthor),
		},
		liblog.Colors["G1"],
	)

	config := new(Config)
	defaultConfig := new(Config)
	defaultConfig.ProxyRotator = libproxyrotator.DefaultConfig
	defaultConfig.Inject = libinject.DefaultConfig
	defaultConfig.Inject.Type = 3
	defaultConfig.Inject.Rules = map[string][]string{
		"akamai.net:80": []string{
			"video.iflix.com",
			"videocdn-2.iflix.com",
			"iflix-videocdn-p1.akamaized.net",
			"iflix-videocdn-p2.akamaized.net",
			"iflix-videocdn-p3.akamaized.net",
			"iflix-videocdn-p6.akamaized.net",
			"iflix-videocdn-p7.akamaized.net",
			"iflix-videocdn-p8.akamaized.net",
		},
	}
	defaultConfig.Inject.ProxyPayload = ""
	defaultConfig.Inject.ProxyTimeout = 5
	defaultConfig.PsiphonCore = 4
	defaultConfig.Psiphon = libpsiphon.DefaultConfig

	libutils.JsonReadWrite(libutils.RealPath("config.json"), config, defaultConfig)

	ProxyRotator := new(libproxyrotator.ProxyRotator)
	ProxyRotator.Config = config.ProxyRotator

	Inject := new(libinject.Inject)
	Inject.Redsocks = Redsocks
	Inject.Config = config.Inject

	go ProxyRotator.Start()
	go Inject.Start()

	time.Sleep(200 * time.Millisecond)

	liblog.LogInfo("Domain Fronting running on port " + Inject.Config.Port, "INFO", liblog.Colors["G1"])
	liblog.LogInfo("Proxy Rotator running on port " + ProxyRotator.Config.Port, "INFO", liblog.Colors["G1"])

	if _, err := os.Stat(libutils.RealPath(config.Psiphon.CoreName)); os.IsNotExist(err) {
		liblog.LogInfo(fmt.Sprintf(
				"Exception:\n\n" +
					"|   File '%s' not exist!\n" +
					"|   Exiting...\n" +
					"|\n",
				config.Psiphon.CoreName),
			"INFO", liblog.Colors["R1"],
		)
		return
	}

	Redsocks.Config = libredsocks.DefaultConfig
	Redsocks.Start()

	for i := 1; i <= config.PsiphonCore; i++ {
		Psiphon := new(libpsiphon.Psiphon)
		Psiphon.Config = config.Psiphon
		Psiphon.ProxyPort = Inject.Config.Port
		Psiphon.KuotaData = libpsiphon.DefaultKuotaData
		Psiphon.ListenPort = libutils.Atoi(ProxyRotator.Config.Port) + i

		go Psiphon.Start(ProxyRotator)
	}

	InterruptHandler.Wait()
}
