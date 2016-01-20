package bootstrap

import (
	"time"
	"log"

	"github.com/go-ini/ini"
	"strings"

	loglocal "zooinit/log"
)

//This basic discovery service bootstrap env info
type envInfo struct {
	//service name, also use for log
	service       string
	discoveryHost string
	discoveryPort string

	//cluster totol qurorum
	qurorum       int
	//sec unit
	timeout       time.Duration

	logPath       string

	//Logger instance for service
	logger        *log.Logger
}

func NewEnvInfo(iniobj *ini.File) (*envInfo) {
	obj := new(envInfo)

	sec := iniobj.Section(CONFIG_SECTION)
	obj.service = sec.Key("service").String()
	if obj.service == "" {
		log.Fatalln("Config of service is empty.")
	}

	discovery := sec.Key("discovery").String()
	if discovery == "" {
		log.Fatalln("Config of discovery is empty.")
	}
	if strings.Index(discovery, ":") == -1 {
		log.Fatalln("Config of discovery need ip:port format.")
	}
	obj.discoveryHost = discovery[0:strings.Index(discovery, ":")]
	obj.discoveryPort = discovery[strings.Index(discovery, ":") + 1:]

	qurorum, err := sec.Key("qurorum").Int()
	if err != nil {
		log.Fatalln("Config of qurorum is error:", err)
	}
	if qurorum < 3 {
		log.Fatalln("Config of qurorum must >=3")
	}
	obj.qurorum = qurorum

	timeout, err := sec.Key("timeout").Float64()
	if err != nil {
		log.Fatalln("Config of timeout is error:", err)
	}
	obj.timeout = time.Duration(int(timeout * 1000000000))

	obj.logPath = sec.Key("log.path").String()
	if obj.logPath == "" {
		log.Fatalln("Config of log.path is empty.")
	}

	obj.logger=loglocal.GetFileLogger(loglocal.GenerateFileLogPathName(obj.logPath, obj.service))
	obj.logger.Println("Configure file parsed. Waiting to be boostrapped.")

	return obj
}
