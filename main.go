package main

import (
	"os"
	"fmt"
	"sync"
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
	appVersionCode = "200125"

	copyrightYear = "2020"
	copyrightAuthor = "Aztec Rabbit"
)

var (
	Redsocks = new(libredsocks.Redsocks)
)

type Config struct {
	ProxyRotator *libproxyrotator.Config
	Inject *libinject.Config
	PsiphonCore int
	Psiphon *libpsiphon.Config
}

func init() {
	InterruptHandler := &libutils.InterruptHandler{
		Handle: func() {
			libpsiphon.Stop()
			libredsocks.Stop(Redsocks)
			liblog.LogKeyboardInterrupt()
		},
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
	defaultConfig.Inject.Proxies = map[string][]string{
		"www.pubgmobile.com:80": []string{
			"akamai.net:80",
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
	Inject.Config = config.Inject
	Inject.Redsocks = Redsocks

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

	// Defined on global variables
	Redsocks.Config = libredsocks.DefaultConfig
	if Redsocks.CheckIsEnabled() {
		liblog.LogInfo("Redsocks started", "INFO", liblog.Colors["G1"])
	}
	Redsocks.Start()

	var wg sync.WaitGroup

	for i := 1; i <= config.PsiphonCore; i++ {
		wg.Add(1)

		Psiphon := new(libpsiphon.Psiphon)
		Psiphon.Config = config.Psiphon
		Psiphon.ProxyPort = Inject.Config.Port
		Psiphon.KuotaData = libpsiphon.KuotaDataDefault
		Psiphon.ListenPort = libutils.Atoi(ProxyRotator.Config.Port) + i

		go Psiphon.Start(&wg, ProxyRotator)
	}

	wg.Wait()
}
