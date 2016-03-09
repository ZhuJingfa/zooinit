package cluster

import (
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-ini/ini"

	"zooinit/config"
	loglocal "zooinit/log"
	"zooinit/utility"
)

// This cluster service bootstrap env info
type envInfo struct {
	// service name, also use for log
	service string
	// Cluster power backend
	clusterBackend string

	// Bootstrap etcd cluster service for boot other cluster service.
	discoveryMethod string
	discoveryTarget string
	discoveryPath   string

	// cluster totol qurorum
	qurorum int
	// sec unit
	timeout time.Duration

	logPath string

	// Logger instance for service
	logger *loglocal.BufferedFileLogger

	// localIP for boot
	localIP net.IP
	// Ip hint use to found which ip for boot bind
	iphint string

	// Health check interval, default 2 sec, same to zookeeper ticktime.
	healthCheckInterval time.Duration

	// boot event related
	eventOnPreRegist       string
	eventOnPostRegist      string
	eventOnReachQurorumNum string
	eventOnPreStart        string
	eventOnStart           string
	eventOnPostStart       string
	eventOnClusterBooted   string
	eventOnHealthCheck     string

	// app start up configuration, app can fetch through env variables
	config map[string]string
}

// New env from file
func NewEnvInfoFile(fname string, backend, servie string) *envInfo {
	iniobj := config.GetConfigInstance(fname)

	return NewEnvInfo(iniobj, backend, servie)
}

func NewEnvInfo(iniobj *ini.File, backend, servie string) *envInfo {
	obj := new(envInfo)

	// init map
	obj.config = make(map[string]string)

	clusterSection := CONFIG_SECTION + "." + backend
	sec, err := iniobj.GetSection(clusterSection)
	if err != nil {
		log.Fatalln("Config of section: " + clusterSection + " is not well configured.")
	}

	obj.service = servie
	if obj.service == "" {
		log.Fatalln("Config of service is empty.")
	}

	obj.logPath = sec.Key("log.path").String()
	if obj.logPath == "" {
		log.Fatalln("Config of log.path is empty.")
	}

	obj.clusterBackend = sec.Key("cluster.backend").String()
	if obj.clusterBackend == "" {
		log.Fatalln("Config of cluster.backend is empty.")
	}
	obj.logger = loglocal.GetConsoleFileMultiLogger(loglocal.GenerateFileLogPathName(obj.logPath, obj.service))
	//flush last log info
	defer obj.logger.Sync()

	//register signal watcher
	obj.registerSignalWatch()

	obj.logger.Println("Service name of cluster is:", obj.service)

	obj.discoveryMethod = sec.Key("discover.method").String()
	if obj.discoveryMethod == "" {
		obj.logger.Fatalln("Config of discover.method is empty.")
	}

	obj.discoveryTarget = sec.Key("discover.target").String()
	if obj.discoveryTarget == "" {
		obj.logger.Fatalln("Config of discover.target is empty.")
	}

	obj.discoveryPath = sec.Key("discover.path").String()
	if obj.discoveryPath == "" {
		obj.logger.Fatalln("Config of discover.path is empty.")
	}
	obj.discoveryPath = obj.discoveryPath + "/" + obj.service

	qurorum, err := sec.Key("qurorum").Int()
	if err != nil {
		obj.logger.Fatalln("Config of qurorum is error:", err)
	}
	if qurorum < 3 {
		obj.logger.Fatalln("Config of qurorum must >=3")
	}
	obj.qurorum = qurorum

	timeout, err := sec.Key("timeout").Float64()
	if err != nil {
		obj.logger.Fatalln("Config of timeout is error:", err)
	}
	if timeout == 0 {
		obj.timeout = CLUSTER_BOOTSTRAP_TIMEOUT
	} else {
		obj.timeout = time.Duration(int(timeout * 1000000000))
	}

	checkInterval, err := sec.Key("health.check.interval").Float64()
	if err != nil {
		obj.logger.Fatalln("Config of health.check.interval is error:", err)
	}
	if checkInterval > 60 || checkInterval < 1 {
		obj.logger.Fatalln("Config of health.check.interval must be between 1-60 sec.")
	}
	if checkInterval == 0 {
		obj.healthCheckInterval = CLUSTER_HEALTH_CHECK_INTERVAL
	} else {
		obj.healthCheckInterval = time.Duration(int(checkInterval * 1000000000))
	}

	// Event process
	obj.eventOnPreRegist = sec.Key("EVENT_ON_PRE_REGIST").String()
	if obj.eventOnPreRegist != "" {
		obj.logger.Println("Found event EVENT_ON_PRE_REGIST:", obj.eventOnPreRegist)
	}
	obj.eventOnPostRegist = sec.Key("EVENT_ON_POST_REGIST").String()
	if obj.eventOnPostRegist != "" {
		obj.logger.Println("Found event EVENT_ON_POST_REGIST:", obj.eventOnPostRegist)
	}
	obj.eventOnReachQurorumNum = sec.Key("EVENT_ON_REACH_QURORUM_NUM").String()
	if obj.eventOnReachQurorumNum != "" {
		obj.logger.Println("Found event EVENT_ON_REACH_QURORUM_NUM:", obj.eventOnReachQurorumNum)
	}
	obj.eventOnPreStart = sec.Key("EVENT_ON_PRE_START").String()
	if obj.eventOnPreStart != "" {
		obj.logger.Println("Found event EVENT_ON_PRE_START:", obj.eventOnPreStart)
	}
	//required
	obj.eventOnStart = sec.Key("EVENT_ON_START").String()
	if obj.eventOnStart == "" {
		obj.logger.Fatalln("Config of EVENT_ON_START is empty.")
	} else {
		obj.logger.Println("Found event EVENT_ON_START:", obj.eventOnStart)
	}
	obj.eventOnPostStart = sec.Key("EVENT_ON_POST_START").String()
	if obj.eventOnPostStart == "" {
		obj.logger.Fatalln("Config of EVENT_ON_POST_START is empty.")
	} else {
		obj.logger.Println("Found event EVENT_ON_POST_START:", obj.eventOnPostStart)
	}
	obj.eventOnClusterBooted = sec.Key("EVENT_ON_CLUSTER_BOOTED").String()
	if obj.eventOnClusterBooted != "" {
		obj.logger.Println("Found event EVENT_ON_CLUSTER_BOOTED:", obj.eventOnClusterBooted)
	}
	obj.eventOnHealthCheck = sec.Key("EVENT_ON_HEALTH_CHECK").String()
	if obj.eventOnHealthCheck == "" {
		obj.logger.Fatalln("Config of EVENT_ON_HEALTH_CHECK is empty.")
	} else {
		obj.logger.Println("Found event EVENT_ON_HEALTH_CHECK:", obj.eventOnHealthCheck)
	}

	obj.iphint = sec.Key("ip.hint").String()
	if obj.iphint == "" {
		obj.logger.Fatalln("Config of ip.hint is empty.")
	}

	// Find localip
	localip, err := utility.GetLocalIPWithIntranet(obj.iphint)
	if err != nil {
		obj.logger.Fatalln("utility.GetLocalIPWithIntranet Please check configuration of discovery is correct.")
	}
	obj.localIP = localip
	obj.logger.Println("Found localip for boot:", obj.localIP)

	// store app config, optional
	appSection := clusterSection + ".config"
	secApp, err := iniobj.GetSection(appSection)
	if err != nil {
		obj.logger.Println("Config of app config section: " + appSection + " is not well configured, continue...")
	} else {
		obj.config = secApp.KeysHash()
		if len(obj.config) > 0 {
			obj.logger.Println("Fetch app config section " + appSection + " KV values:")

			for key, value := range obj.config {
				obj.logger.Println("Key:", key, " Value:", value)
			}
		} else {
			obj.logger.Println("Fetch app config section: empty")
		}
	}

	obj.logger.Println("Configure file parsed. Waiting to be boostrapped...")

	return obj
}

func (e *envInfo) GetQurorum() int {
	if e == nil {
		return 0
	}

	return e.qurorum
}

func (e *envInfo) GetTimeout() time.Duration {
	if e == nil {
		return 0
	}

	return e.timeout
}

func (e *envInfo) Service() string {
	if e == nil {
		return ""
	}

	return e.service
}

func (e *envInfo) Logger() *loglocal.BufferedFileLogger {
	if e == nil {
		return nil
	}

	return e.logger
}

func (e *envInfo) GetNodename() string {
	if e == nil {
		return ""
	}

	return e.clusterBackend + "-" + e.localIP.String()
}

func (e *envInfo) registerSignalWatch() {
	if e == nil {
		return
	}

	sg := utility.NewSignalCatcher()
	call := utility.NewSignalCallback(func(sig os.Signal, data interface{}) {
		e.logger.Println("Receive signal: " + sig.String() + " App will terminate, bye.")
		e.logger.Sync()
	}, nil)

	sg.SetDefault(call)
	sg.EnableExit()
	e.logger.Println("Init System SignalWatcher, catch list:", strings.Join(sg.GetSignalStringList(), ", "))

	sg.RegisterAndServe()
}
