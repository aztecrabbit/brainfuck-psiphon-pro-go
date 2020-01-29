package libpsiphon

import (
	"os"
	"os/exec"
	"fmt"
	"time"
	"bufio"
	"strconv"
	"strings"
	"syscall"
	"encoding/json"

	"github.com/aztecrabbit/liblog"
	"github.com/aztecrabbit/libutils"
	"github.com/aztecrabbit/libproxyrotator"
)

var (
	Loop = true
	DefaultConfig = &Config{
		CoreName: "psiphon-tunnel-core",
		Tunnel: 1,
		Region: "SG",
		Protocols: []string{
			"FRONTED-MEEK-HTTP-OSSH",
			"FRONTED-MEEK-OSSH",
		},
		TunnelWorkers: 4,
		KuotaDataLimit: 4000000,
		Authorizations: make([]string, 0),
	}
	DefaultKuotaData = &KuotaData{
		Port: make(map[int]map[string]float64),
		All: 0,
	}
	PsiphonDirectory = libutils.RealPath("storage/psiphon")
)

func Stop() {
	Loop = false
	os.RemoveAll(PsiphonDirectory + "/data")
}

type Config struct {
	CoreName string
	Tunnel int
	Region string
	Protocols []string
	TunnelWorkers int
	KuotaDataLimit int
	Authorizations []string
}

type KuotaData struct {
	Port map[int]map[string]float64
	All float64
}

type Data struct {
	UpstreamProxyURL string
	DataStoreDirectory string
	LocalSocksProxyPort int
	SponsorId string
	PropagationChannelId string
	EmitBytesTransferred bool
	EmitDiagnosticNotices bool
	DisableLocalHTTPProxy bool
	EgressRegion string
	TunnelPoolSize int
	ConnectionWorkerPoolSize int
	LimitTunnelProtocols []string
	Authorizations []string
}

type Psiphon struct {
	ProxyRotator *libproxyrotator.ProxyRotator
	Config *Config
	ProxyPort string
	KuotaData *KuotaData
	ListenPort int
	TunnelConnected int
}

func (p *Psiphon) LogInfo(message string, color string) {
	if Loop {
		liblog.LogInfo(message, strconv.Itoa(p.ListenPort), color)
	}
}

func (p *Psiphon) GetAuthorizations() []string {
	data := make([]string, 0)

	if len(p.Config.Authorizations) != 0 {
		data = append(data, p.Config.Authorizations[0])
		p.Config.Authorizations = append(p.Config.Authorizations[1:], p.Config.Authorizations[0])
	}

	return data
}

func (p *Psiphon) CheckKuotaDataLimit(sent float64, received float64) bool {
	if p.Config.KuotaDataLimit != 0 && int(p.KuotaData.Port[p.ListenPort]["all"]) >= p.Config.KuotaDataLimit &&
			int(sent) == 0 && int(received) <= 64000 {
		return false
	}

	return true
}

func (p *Psiphon) Start() {
	PsiphonData := &Data{
		UpstreamProxyURL: "http://127.0.0.1:" + p.ProxyPort,
		DataStoreDirectory: PsiphonDirectory + "/data/" + strconv.Itoa(p.ListenPort),
		LocalSocksProxyPort: p.ListenPort,
		SponsorId: "0000000000000000",
		PropagationChannelId: "0000000000000000",
		EmitBytesTransferred: true,
		EmitDiagnosticNotices: true,
		DisableLocalHTTPProxy: true,
		EgressRegion: strings.ToUpper(p.Config.Region),
		TunnelPoolSize: p.Config.Tunnel,
		ConnectionWorkerPoolSize: p.Config.TunnelWorkers,
		LimitTunnelProtocols: p.Config.Protocols,
		Authorizations: p.GetAuthorizations(),
	}

	libutils.JsonWrite(PsiphonData, PsiphonData.DataStoreDirectory + "/config.json")
	libutils.CopyFile(
		PsiphonDirectory + "/database/psiphon.boltdb",
		PsiphonData.DataStoreDirectory + "/psiphon.boltdb",
		0666,
	)

	p.LogInfo("Connecting", liblog.Colors["G1"])

	for Loop {
		p.KuotaData.Port[p.ListenPort] = make(map[string]float64)
		p.KuotaData.Port[p.ListenPort]["all"] = 0
		p.TunnelConnected = 0

		command := exec.Command(
			libutils.RealPath(p.Config.CoreName), "-config", PsiphonData.DataStoreDirectory + "/config.json",
		)

		stderr, err := command.StderrPipe()
		if err != nil {
			panic(err)
		}

		scanner := bufio.NewScanner(stderr)
		go func() {
			var text string
			var line map[string]interface{}
			for Loop && scanner.Scan() {
				text = scanner.Text()
				json.Unmarshal([]byte(text), &line)

				noticeType := line["noticeType"]

				if noticeType == "BytesTransferred" {
					data := line["data"].(map[string]interface{})
					diagnosticId := data["diagnosticID"].(string)
					sent := data["sent"].(float64)
					received := data["received"].(float64)

					p.KuotaData.Port[p.ListenPort][diagnosticId] += sent + received
					p.KuotaData.Port[p.ListenPort]["all"] += sent + received
					p.KuotaData.All += sent + received

					if p.CheckKuotaDataLimit(sent, received) == false {
						break
					}

					liblog.LogReplace(fmt.Sprintf(
							"%v (%v) (%v) (%v)",
							p.ListenPort,
							diagnosticId,
							libutils.BytesToSize(p.KuotaData.Port[p.ListenPort][diagnosticId]),
							libutils.BytesToSize(p.KuotaData.All),
						),
						liblog.Colors["G1"],
					)

				} else if noticeType == "ActiveTunnel" {
					p.ProxyRotator.AddProxy("0.0.0.0:" + strconv.Itoa(p.ListenPort))
					p.TunnelConnected++
					if p.Config.Tunnel > 1 {
						diagnosticId := line["data"].(map[string]interface{})["diagnosticID"].(string)
						p.LogInfo(fmt.Sprintf("Connected (%s)", diagnosticId), liblog.Colors["Y1"])
					}
					if p.TunnelConnected == p.Config.Tunnel {
						p.LogInfo("Connected", liblog.Colors["Y1"])
					}

				} else if noticeType == "Alert" {
					message := line["data"].(map[string]interface{})["message"].(string)

					if strings.Contains(message, "meek round trip failed") {
						if p.TunnelConnected == p.Config.Tunnel && (
								message == "meek round trip failed: remote error: tls: bad record MAC" ||
								message == "meek round trip failed: context deadline exceeded" ||
								message == "meek round trip failed: EOF" ||
								strings.Contains(message, "psiphon.CustomTLSDial")) {
							// p.LogInfo(message, liblog.Colors["R1"])
							break
						}
					} else if strings.Contains(message, "controller shutdown due to component failure") ||
							strings.Contains(message, "psiphon.(*ServerContext).DoStatusRequest") ||
							strings.Contains(message, "psiphon.(*Tunnel).sendSshKeepAlive") ||
							strings.Contains(message, "psiphon.(*Tunnel).Activate") ||
							strings.Contains(message, "underlying conn is closed") ||
							strings.Contains(message, "duplicate tunnel:") ||
							strings.Contains(message, "tunnel failed:") {
						// p.LogInfo("Break: " + text, liblog.Colors["R1"])
						break
					} else if strings.Contains(message, "A connection attempt failed because the connected party did not properly respond after a period of time") ||
							strings.Contains(message, "No connection could be made because the target machine actively refused it") ||
							strings.Contains(message, "tunnel.dialTunnel: dialConn is not a Closer") ||
							strings.Contains(message, "psiphon.(*ServerContext).DoConnectedRequest") ||
							strings.Contains(message, "making proxy request: unexpected EOF") ||
							strings.Contains(message, "psiphon.(*MeekConn).readPayload") ||
							strings.Contains(message, "response status: 403 Forbidden") ||
							strings.Contains(message, "meek connection has closed") ||
							strings.Contains(message, "meek connection is closed") ||
							strings.Contains(message, "psiphon.(*MeekConn).relay") ||
							strings.Contains(message, "unexpected status code:") ||
							strings.Contains(message, "RemoteAddr returns nil") ||
							strings.Contains(message, "network is unreachable") ||
							strings.Contains(message, "close tunnel ssh error") ||
							strings.Contains(message, "tactics request failed") ||
							strings.Contains(message, "API request rejected") ||
							strings.Contains(message, "context canceled") ||
							strings.Contains(message, "no such host") {
						continue
					} else if strings.Contains(message, "bind: address already in use") {
						p.LogInfo("Port already in use", liblog.Colors["R1"])
						break
					} else {
						p.LogInfo(text, liblog.Colors["R1"])
					}

				} else if noticeType == "UpstreamProxyError" {
					continue

				} else {
					// liblog.LogInfoSplit(text, 22, "INFO", liblog.Colors["G2"])

				}
			}

			command.Process.Signal(syscall.SIGTERM)
		}()

		command.Start()
		command.Wait()

		p.ProxyRotator.DeleteProxy("0.0.0.0:" + strconv.Itoa(p.ListenPort))

		time.Sleep(200 * time.Millisecond)

		p.LogInfo(fmt.Sprintf("Reconnecting (%s)", libutils.BytesToSize(p.KuotaData.Port[p.ListenPort]["all"])), liblog.Colors["G1"])
	}
}
