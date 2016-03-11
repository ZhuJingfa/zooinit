package bootstrap

import (
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"github.com/coreos/etcd/client"

	"zooinit/cluster/etcd"
	"zooinit/log"
	"zooinit/utility"
)

const (
	// INTERNAL discovery findpath
	INTERNAL_FINDPATH         = "/zooinit/boot"
	CLUSTER_BOOTSTRAP_TIMEOUT = 5 * time.Minute
)

func BootstrapEtcd(env *envInfo) error {
	// flush last log info
	defer env.logger.Sync()
	env.logger.Println("Starting to boot Etcd...")

	// Internal discovery service
	internalClientUrl := "http://" + env.internalHost + ":" + env.internalPort
	// Api to internal service
	api, err := etcd.NewApiKeys([]string{internalClientUrl})
	if err != nil {
		env.logger.Fatal("Etcd NewApi error:", err)
	}

	// Need declare for terminate
	var internalCmd *exec.Cmd
	if env.isSelfIp {
		internalPeerUrl := "http://" + env.internalHost + ":" + env.internalPeer

		env.logger.Println("Etcd Internal PeerUrl:", internalPeerUrl)
		env.logger.Println("Etcd Internal ClientUrl:", internalClientUrl)

		// Add & can't fast wait
		// data-dir can't be same with discovery service.
		intName := "etcd.initial"
		intExecCmd := "etcd --data-dir " + env.internalDataDir + " -wal-dir " + env.internalWalDir + " -name " + intName +
			" -initial-advertise-peer-urls " + internalPeerUrl +
			" -listen-peer-urls " + internalPeerUrl +
			" -listen-client-urls " + internalClientUrl +
			" -advertise-client-urls " + internalClientUrl +
			" -initial-cluster " + intName + "=" + internalPeerUrl

		env.logger.Println("Etcd Internal ExecCmd:", intExecCmd)

		// Boot internal discovery service
		path, args, err := utility.ParseCmdStringWithParams(intExecCmd)
		if err != nil {
			env.logger.Fatalln("Error ParseCmdStringWithParams internal service:", err)
		}

		internalCmd = exec.Command(path, args...)
		loggerIOAdapter := log.NewLoggerIOAdapter(env.logger)
		loggerIOAdapter.SetPrefix("Etcd internal server: ")
		internalCmd.Stdout = loggerIOAdapter
		internalCmd.Stderr = loggerIOAdapter
		err = internalCmd.Start()

		if err != nil {
			env.logger.Fatalln("Exec Internal ExecCmd Error:", err)
		} else {
			env.logger.Println("Exec Internal OK, PID:", internalCmd.Process.Pid)

			// Release process after cluster up.
			// may runtime error: invalid memory address or nil pointer dereference
			defer func() {
				if internalCmd.Process != nil {
					internalCmd.Process.Kill()
				}
			}()

			// Set PID
			env.internalCmdInstance = internalCmd
			env.logger.Println("Internal service started.")

			// Important!!! check upstarted
			env.logger.Println("Etcd LoopTimeoutRequest for check internal's startup...")

			internalCheckout := 3 * time.Second
			isHealth, err := LoopTimeoutRequest(internalCheckout, env, func() (bool, error) {
				return etcd.CheckHealth(internalClientUrl)
			})
			if err != nil {
				env.logger.Fatal("Error check internal error: ", err)
			} else if isHealth != true {
				env.logger.Fatal("Error check internal server health: ", isHealth)
				env.logger.Fatal("Cluster bootstrap faild: failed to bootstrap in ", internalCheckout.String())
			}

			resp, err := http.Get(internalClientUrl + "/v2/stats/self")
			if err != nil {
				env.logger.Fatal("Error fetch stats self: ", err)
			}
			env.logger.Println("Etcd internal Stat self: ", resp)

			_, err = api.Conn().Delete(etcd.Context(), INTERNAL_FINDPATH, &client.DeleteOptions{Dir: true, Recursive: true})
			if err != nil {
				//type safe cast
				err, ok := err.(client.Error)
				if ok && err.Code != client.ErrorCodeKeyNotFound {
					env.logger.Fatal("Delete ", INTERNAL_FINDPATH, " error:", err)
				}
			}

			env.logger.Println("Set Cluster bootstrap timeout:", env.timeout.String())
			_, err = api.Conn().Set(etcd.Context(), INTERNAL_FINDPATH, "", &client.SetOptions{TTL: env.timeout, Dir: true})
			if err != nil {
				env.logger.Fatal("Set TTL for ", INTERNAL_FINDPATH, " error:", err)
			}

			env.logger.Println("Set Qurorum ", INTERNAL_FINDPATH+"/_config/size to ", env.qurorum)
			_, err = api.Conn().Set(etcd.Context(), INTERNAL_FINDPATH+"/_config/size", strconv.Itoa(env.qurorum), nil)
			if err != nil {
				env.logger.Fatal("Set Qurorum ", INTERNAL_FINDPATH+"/_config/size error: ", err)
			}
		}
	}

	// Cluster member startup info
	discoveryPeerUrl := "http://" + env.localIP.String() + ":" + env.discoveryPeer
	discoveryClientUrl := "http://" + env.localIP.String() + ":" + env.discoveryPort
	env.logger.Println("Etcd Discovery PeerUrl:", discoveryPeerUrl)
	env.logger.Println("Etcd Discovery ClientUrl:", discoveryClientUrl)

	disExecCmd := env.cmd + " --data-dir " + env.cmdDataDir + " -wal-dir " + env.cmdWalDir +
		" -snapshot-count " + strconv.Itoa(env.cmdSnapCount) +
		" -name " + "etcd.bootstrap." + env.localIP.String() +
		" -initial-advertise-peer-urls " + discoveryPeerUrl +
		" -listen-peer-urls " + discoveryPeerUrl +
		" -listen-client-urls http://127.0.0.1:2379," + discoveryClientUrl +
		" -advertise-client-urls " + discoveryClientUrl +
		" -discovery " + internalClientUrl + "/v2/keys" + INTERNAL_FINDPATH

	env.logger.Println("Etcd Discovery ExecCmd: ", disExecCmd)

	// Boot internal discovery service
	// Need to rm -rf /tmp/etcd/ because dir may be used before
	path, args, err := utility.ParseCmdStringWithParams(disExecCmd)
	if err != nil {
		env.logger.Fatalln("Error ParseCmdStringWithParams cluster bootstrap: ", err)
	}

	clusterCmd := exec.Command(path, args...)
	loggerIOAdapter := log.NewLoggerIOAdapter(env.logger)
	loggerIOAdapter.SetPrefix("Etcd discovery member: ")
	clusterCmd.Stdout = loggerIOAdapter
	clusterCmd.Stderr = loggerIOAdapter

	err = clusterCmd.Start()
	// may runtime error: invalid memory address or nil pointer dereference
	defer func() {
		if clusterCmd.Process != nil {
			clusterCmd.Process.Kill()
		}
	}()

	if err != nil {
		env.logger.Fatalln("Exec Discovery ExecCmd Error: ", err)
	} else {
		env.logger.Println("Exec Discovery Etcd member OK, PID: ", clusterCmd.Process.Pid)
		env.logger.Println("Etcd member service ", discoveryClientUrl, " started,  waiting to be bootrapped.")
	}

	// Important!!! check upstarted
	env.logger.Println("Etcd LoopTimeoutRequest for check discovery cluster's startup...")
	isHealth, err := LoopTimeoutRequest(env.timeout, env, func() (bool, error) {
		return etcd.CheckHealth(discoveryClientUrl)
	})
	if err != nil {
		env.logger.Fatal("Error check discovery error: ", err)
	} else if isHealth != true {
		env.logger.Fatal("Error check discovery server health: ", isHealth)
		env.logger.Fatal("Cluster bootstrap faild: failed to bootstrap in ", env.timeout.String())
	}

	// Close internal service
	env.logger.Println("Cluster etcd service is booted. Internal service is going to be terminated.")
	if internalCmd != nil && internalCmd.Process != nil {
		internalCmd.Process.Kill()
	}

	// Watch dog run
	env.logger.Println("Cluster watch dog is going to run...")
	w := NewWatchDog(env, internalClientUrl, discoveryClientUrl)
	go w.Run()

	// check cluster bootstraped and register memberself
	// If stoped, process's output can't trace no longer
	clusterCmd.Wait()

	return nil
}

//request until sucess
func LoopTimeoutRequest(timeout time.Duration, env *envInfo, routine func() (result bool, err error)) (result bool, err error) {
	var charlist []byte

	//flush last log info
	defer env.logger.Sync()

	result = false
	start := time.Now()
	for {
		result, err = routine()
		if !result || err != nil {
			charlist = append(charlist, byte('.'))
			// sleep 100ms
			end := time.Now()
			// not time outed
			if end.Sub(start) < timeout {
				time.Sleep(100 * time.Millisecond)
			} else {
				break
			}
		} else {
			break
		}
	}

	env.logger.Println("Fetched data LoopTimeoutRequest for loop:", string(charlist))

	return result, err
}
