package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const kindaSdnHttpAddr = "http://127.0.0.1:9002"

type Cleaner func()

var expectCallback atomic.Bool

func runCallbackService() chan int {
	callbackChan := make(chan int)
	http.HandleFunc("/done", func(w http.ResponseWriter, r *http.Request) {
		updateTime, _ := strconv.Atoi(r.URL.Query()["etcd-update-time"][0])
		if expectCallback.Load() {
			callbackChan <- updateTime
			expectCallback.Store(false)
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("OK"))
	})
	go func() {
		err := http.ListenAndServe("127.0.0.1:16423", nil)
		if err != nil {
			panic(err)
		}
	}()
	return callbackChan
}

func runKind() Cleaner {
	output, err := exec.Command("kind", "create", "cluster").CombinedOutput()
	ostr := string(output)
	_ = ostr
	if err != nil || !strings.Contains(string(output), "You can now use your cluster with"){
		panic(err)
	}
	sdnShimDir := "/home/flok3n/develop/k8s_inc/src/sdn-shim"
	neededLines := []string{
		"customresourcedefinition.apiextensions.k8s.io/incswitches.inc.kntp.com created",
		"customresourcedefinition.apiextensions.k8s.io/p4programs.inc.kntp.com created",
		"customresourcedefinition.apiextensions.k8s.io/sdnshims.inc.kntp.com created",
		"customresourcedefinition.apiextensions.k8s.io/topologies.inc.kntp.com created",
	}
	output, err = exec.Command("make", "--directory", sdnShimDir, "generate", "manifests", "install").CombinedOutput()
	if err != nil {
		panic(err)
	}
	for _, line := range neededLines {
		if !strings.Contains(string(output), line) {
			panic("missing line")
		}
	}
	return func() {
		_, err := exec.Command("kind", "delete", "cluster").Output()
		if err != nil {
			panic(err)
		}
	}
}

func runKindaSdn(k int, incSwitchFraction float32) Cleaner {
	path := "/home/flok3n/develop/k8s_inc/src/kinda-sdn/kinda-sdn"
	kindaSdnCmd := exec.Command(path,
		"--evaluate-reconciliation",
	 	fmt.Sprintf("--fat-tree-k=%d", k),
		fmt.Sprintf("--inc-switch-fraction=%f", incSwitchFraction),
	)
	if err := kindaSdnCmd.Start(); err != nil {
		panic(err)
	}
	for {
		r, err := http.Get(kindaSdnHttpAddr + "/health")
		if err == nil && r.StatusCode == 200 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return func() {
		kindaSdnCmd.Process.Signal(os.Interrupt)
		kindaSdnCmd.Wait()
	}
}

func createShimResource() Cleaner {
	exec.Command("kubectl", "create", "-f", "./shim.yaml").Output()
	for {
		output, err := exec.Command("kubectl", "get", "sdnshim", "kinda-sdn-shim", "-o=jsonpath='{.status.conditions}'").Output()
		if err == nil {
			if strings.Count(string(output), "\"status\":\"True\"") == 2 {
				break
			}
		}
		time.Sleep(100*time.Millisecond)
	}
	return func() {
		exec.Command("kubectl", "delete", "-f", "./shim.yaml").Output()
		for {
			output, err := exec.Command("kubectl", "get", "topologies").CombinedOutput()
			if err != nil {
				panic(err)
			}
			if strings.Contains(string(output), "No resources found in default namespace.") {
				break
			}
			time.Sleep(100*time.Millisecond)
		}
	}
}

func runSdnShim() Cleaner {
	path := "/home/flok3n/develop/k8s_inc/src/sdn-shim/sdn-shim"
	sdnShimCmd := exec.Command(path)
	stdOut, _ := sdnShimCmd.StderrPipe()
	if err := sdnShimCmd.Start(); err != nil {
		panic(err)
	}
	buff := make([]byte, 1)
	allStdOut := []byte{}
	for {
		n, err := stdOut.Read(buff)
		if err != nil || n == 0 {
			panic("failed to read stdout")
		}
		allStdOut = append(allStdOut, buff[0])
		if strings.Count(string(allStdOut), "Starting workers") == 2 {
			break
		}
	}
	return func() {
		sdnShimCmd.Process.Signal(os.Interrupt)
		sdnShimCmd.Wait()
	}
}

func addDevice(name string) {
	body := []byte(fmt.Sprintf(`{"devices": [{"name": "%s", "deviceType": "net", "links": ["s-0-0-0"]}]}`, name))
	r, err := http.Post(kindaSdnHttpAddr + "/add-devices", "application/json", bytes.NewBuffer(body))
	if err != nil || r.StatusCode != 200 {
		panic(err)
	}
	r.Body.Close()
}

func removeDevice(name string) {
	body := []byte(fmt.Sprintf(`{"name": "%s"}`, name))
	r, err := http.Post(kindaSdnHttpAddr + "/delete-device", "application/json", bytes.NewBuffer(body))
	if err != nil || r.StatusCode != 200 {
		panic(err)
	}
	r.Body.Close()
}

func changeProgram() {
	body := []byte(`{"deviceName": "s-0-0-1", "programName": "telemetry"}`)
	r, err := http.Post(kindaSdnHttpAddr + "/change-program", "application/json", bytes.NewBuffer(body))
	if err != nil || r.StatusCode != 200 {
		panic(err)
	}
	r.Body.Close()
}

func main() {
	resultsFile := "/home/flok3n/develop/k8s_inc_analysis/data/shim_eval/reconciliation_time_all_diffrent_fraction.csv"
	retries := 3
	minK := 6
	maxK := 30
	incSwitchFractions := []float32{0.2, 0.8}
	incSwitchFractionStrs := []string{"0.2", "0.8"}
	// incSwitchFractions := []float32{0.2, 0.5, 0.8}
	// incSwitchFractionStrs := []string{"0.2", "0.5", "0.8"}
	f, err := os.Create(resultsFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString("mode_key,k,inc_switch_fraction,scenario,topo_update_time_ms,total_time_ms\n")
	scenarios := []string{"change-program"}
	// scenarios := []string{"add-device", "remove-device-inc-switch", "change-program"}
	// scenarios := []string{"add-device", "remove-device-inc-switch", "remove-device-net", "change-program"}
	// scenarios := []string{"change-program"}

	expectCallback.Store(false)
	callbackChan := runCallbackService()

	for k := minK; k <= maxK; k += 2 {
		for i, incSwitchFraction := range incSwitchFractions {
			incSwitchFractionStr := incSwitchFractionStrs[i]
			for _, scenario := range scenarios {
				fmt.Printf("Running for scenario: %s, k: %d, isw: %s\n", scenario, k, incSwitchFractionStr)
				for r := 0; r < retries; r++ {
					func () {
						stopKind := runKind()
						defer stopKind()
						stopKindaSdn := runKindaSdn(k, incSwitchFraction)
						defer stopKindaSdn()
						stopSdnShim := runSdnShim()
						defer stopSdnShim()
						cleanupShimResource := createShimResource()	
						defer cleanupShimResource()
						time.Sleep(2*time.Second)

						expectCallback.Store(true)
						
						startTime := time.Now()
						switch scenario {
						case "add-device":
							addDevice("x1")
						case "remove-device-inc-switch":
							removeDevice("s-0-0-1")
						case "remove-device-net":
							removeDevice("s-0-1-1")
						case "change-program":
							changeProgram()
						default:
							panic("unkown scenario")
						}
						updateTime := <-callbackChan
						reconciliationTime := time.Since(startTime)
						mode_key := fmt.Sprintf("%d-%s-%s", k, incSwitchFractionStr, scenario)
						f.WriteString(fmt.Sprintf("%s,%d,%s,%s,%d,%d\n", 
							mode_key, k, incSwitchFractionStr, scenario, updateTime, reconciliationTime.Milliseconds()))
						time.Sleep(2*time.Second)
					}()
				}
			}
		}
	}
}
