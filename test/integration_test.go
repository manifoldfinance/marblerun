package coordinator

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/edgelesssys/coordinator/coordinator/core"
	"github.com/stretchr/testify/assert"
)

// TODO: Use correct values here
const manifestJSON string = `{
	"Packages": {
		"backend": {
			"Debug": true,
			"SecurityVersion": 1,
			"ProductID": [3]
		},
		"frontend": {
			"Debug": true,
			"SecurityVersion": 2,
			"ProductID": [3]
		}
	},
	"Infrastructures": {
		"Azure": {
			"QESVN": 2,
			"PCESVN": 3,
			"CPUSVN": [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15],
			"RootCA": [3,3,3]
		}
	},
	"Marbles": {
		"test_marble_server": {
			"Package": "backend",
			"Parameters": {
				"Files": {
					"/tmp/defg.txt": "foo",
					"/tmp/jkl.mno": "bar"
				},
				"Argv": [
					"serve"
				],
				"Env": {
					"IS_FIRST": "true",
					"ROOT_CA": "$$root_ca",
					"SEAL_KEY": "$$seal_key",
					"MARBLE_CERT": "$$marble_cert",
					"MARBLE_KEY": "$$marble_key"
			}
			}
		},
		"test_marble_client": {
			"Package": "backend",
			"Parameters": {
				"Files": {
					"/tmp/defg.txt": "foo",
					"/tmp/jkl.mno": "bar"
				},
				"Env": {
					"IS_FIRST": "true",
					"ROOT_CA": "$$root_ca",
					"SEAL_KEY": "$$seal_key",
					"MARBLE_CERT": "$$marble_cert",
					"MARBLE_KEY": "$$marble_key"
			}
			}
		},
		"bad_marble": {
			"Package": "frontend",
			"Parameters": {
				"Files": {
					"/tmp/defg.txt": "foo",
					"/tmp/jkl.mno": "bar"
				},
				"Env": {
					"ROOT_CA": "$$root_ca",
					"SEAL_KEY": "$$seal_key",
					"MARBLE_CERT": "$$marble_cert",
					"MARBLE_KEY": "$$marble_key"
			}
		}
		}
	},
	"Clients": {
		"owner": [9,9,9]
	}
}`

var coordinatorExe = flag.String("c", "", "Coordinator executable")
var marbleExe = flag.String("m", "", "Marble executable")
var simulationMode = flag.Bool("s", false, "Execute test in simulation mode (without real quoting)")
var marbleServerAddr, clientServerAddr string
var manifest core.Manifest

func TestMain(m *testing.M) {
	flag.Parse()
	if *coordinatorExe == "" {
		log.Fatalln("You must provide the path of the coordinator executable using th -c flag.")
	}

	if *marbleExe == "" {
		log.Fatalln("You must provide the path of the marble executable using th -m flag.")
	}

	if _, err := os.Stat(*coordinatorExe); err != nil {
		log.Fatalln(err)
	}
	if _, err := os.Stat(*marbleExe); err != nil {
		log.Fatalln(err)
	}

	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		log.Fatalln(err)
	}

	// get unused ports
	var listenerMarbleAPI, listenerClientAPI net.Listener
	listenerMarbleAPI, marbleServerAddr = getListenerAndAddr()
	listenerClientAPI, clientServerAddr = getListenerAndAddr()
	listenerMarbleAPI.Close()
	listenerClientAPI.Close()
	log.Printf("Got marbleServerAddr: %v and clientServerAddr: %v\n", marbleServerAddr, clientServerAddr)
	os.Exit(m.Run())
}

func getListenerAndAddr() (net.Listener, string) {
	const localhost = "localhost:"

	listener, err := net.Listen("tcp", localhost)
	if err != nil {
		panic(err)
	}

	addr := listener.Addr().String()

	// addr contains IP address, we want hostname
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		panic(err)
	}
	return listener, localhost + port
}

// sanity test of the integration test environment
func TestTest(t *testing.T) {
	assert := assert.New(t)

	cfgFilename := createCoordinatorConfig()
	defer cleanupCoordinatorConfig(cfgFilename)
	assert.Nil(startCoordinator(cfgFilename).Kill())

	marbleCfg := createMarbleConfig(marbleServerAddr, "test_marble_client", "")
	defer cleanupMarbleConfig(marbleCfg)
	assert.Nil(startMarble(marbleCfg).Process.Kill())
}

func TestMarbleAPI(t *testing.T) {
	assert := assert.New(t)

	// start Coordinator
	log.Println("Starting a coordinator enclave")
	cfgFilename := createCoordinatorConfig()
	defer cleanupCoordinatorConfig(cfgFilename)
	coordinatorProc := startCoordinator(cfgFilename)
	assert.NotNil(coordinatorProc)
	defer coordinatorProc.Kill()

	// set Manifest
	log.Println("Setting the Manifest")
	err := setManifest(manifest)
	assert.Nil(err, "failed to set Manifest: %v", err)

	// wait for me
	// marbleCfg := createMarbleConfig(marbleServerAddr, "test_marble_server", "test_marble_server")
	// log.Printf("config; %v", marbleCfg)
	// log.Printf("coordinator Addr: %v", marbleServerAddr)
	// time.Sleep(10000000 * time.Second)

	// start server
	log.Println("Starting a Server-Marble")
	serverCfg := createMarbleConfig(marbleServerAddr, "test_marble_server", "server,backend")
	defer cleanupMarbleConfig(serverCfg)
	serverCmd := runMarble(assert, serverCfg, true, false)
	defer serverCmd.Process.Kill()
	err = waitForServer()
	// start clients
	log.Println("Starting a bunch of Client-Marbles")
	assert.Nil(err, "failed to start server-marble: %v", err)
	clientCfg := createMarbleConfig(marbleServerAddr, "test_marble_client", "client,frontend")
	defer cleanupMarbleConfig(clientCfg)
	_ = runMarble(assert, clientCfg, true, true)
	_ = runMarble(assert, clientCfg, true, true)
	if !*simulationMode {
		// start bad marbles (would be accepted if we run in SimulationMode)
		badCfg := createMarbleConfig(marbleServerAddr, "bad_marble", "bad")
		defer cleanupMarbleConfig(badCfg)
		_ = runMarble(assert, badCfg, false, true)
		_ = runMarble(assert, badCfg, false, true)
	}
}

func TestRestart(t *testing.T) {
	assert := assert.New(t)
	log.Println("Testing the restart capabilities")
	// start Coordinator
	log.Println("Starting a coordinator enclave")
	cfgFilename := createCoordinatorConfig()
	defer cleanupCoordinatorConfig(cfgFilename)
	coordinatorProc := startCoordinator(cfgFilename)
	assert.NotNil(coordinatorProc)
	// set Manifest
	log.Println("Setting the Manifest")
	err := setManifest(manifest)
	assert.Nil(err, "failed to set Manifest: %v", err)
	// start server
	log.Println("Starting a Server-Marble")
	serverCfg := createMarbleConfig(marbleServerAddr, "test_marble_server", "server,backend")
	defer cleanupMarbleConfig(serverCfg)
	serverCmd := runMarble(assert, serverCfg, true, false)
	defer serverCmd.Process.Kill()
	err = waitForServer()
	// start clients
	log.Println("Starting a bunch of Client-Marbles")
	assert.Nil(err, "failed to start server-marble: %v", err)
	clientCfg := createMarbleConfig(marbleServerAddr, "test_marble_client", "client,frontend")
	defer cleanupMarbleConfig(clientCfg)
	_ = runMarble(assert, clientCfg, true, true)
	_ = runMarble(assert, clientCfg, true, true)

	// simulate restart of coordinator
	log.Println("Simulating a restart of the coordinator enclave...")
	log.Println("Killing the old instance")
	if err := coordinatorProc.Kill(); err != nil {
		panic(err)
	}
	log.Println("Restarting the old instance")
	coordinatorProc = startCoordinator(cfgFilename)
	assert.NotNil(coordinatorProc)
	defer coordinatorProc.Kill()

	// try do malicious update of manifest
	log.Println("Trying to set a new Manifest, which should already be set")
	err = setManifest(manifest)
	assert.NotNil(err, "expected updating of manifest to fail, but succeeded")

	// start a bunch of client marbles and assert they still work with old server marble
	log.Println("Starting a bunch of Client-Marbles, which should still authenticate successfully with the Server-Marble")
	_ = runMarble(assert, clientCfg, true, true)
	_ = runMarble(assert, clientCfg, true, true)
}

func runMarble(assert *assert.Assertions, marbleCfg string, shouldSucceed bool, terminates bool) *exec.Cmd {
	log.Println("Starting marble")
	marbleCmd := startMarble(marbleCfg)
	assert.NotNil(marbleCmd)

	if !terminates {
		return marbleCmd
	}

	// Check that Marble Authenticated successfully
	err := marbleCmd.Wait()
	if !shouldSucceed {
		assert.NotNil(err, "expected Wait to fail because of return value != 0, but got not error")
		assert.NotNil(marbleCmd.ProcessState)
		exitCode := marbleCmd.ProcessState.ExitCode()
		assert.NotEqual(0, exitCode, "expected marble authentication to fail, but got exit code: %v", exitCode)
		return marbleCmd
	}
	assert.Nil(err, "error while waiting for marble process: %v", err)
	assert.NotNil(marbleCmd.ProcessState, "empty ProcessState after Wait")
	exitCode := marbleCmd.ProcessState.ExitCode()
	assert.Equal(0, exitCode, "marble authentication failed. exit code: %v", exitCode)
	if exitCode == 0 {
		log.Println("Marble authenticated successfully and terminated.")
	}
	return marbleCmd
}

func waitForServer() error {
	log.Println("Waiting for server...")
	timeout := time.Second * 5
	var err error
	for i := 0; i < 20; i++ {
		var conn net.Conn
		conn, err = net.DialTimeout("tcp", net.JoinHostPort("localhost", "8080"), timeout)
		if err == nil {
			conn.Close()
			log.Println("Server started")
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("connection error: %v", err)
}

func TestClientAPI(t *testing.T) {
	assert := assert.New(t)
	eof := errors.New("EOF")

	// start Coordinator
	cfgFilename := createCoordinatorConfig()
	defer cleanupCoordinatorConfig(cfgFilename)
	coordinatorProc := startCoordinator(cfgFilename)
	assert.NotNil(coordinatorProc, "could not start coordinator")
	defer coordinatorProc.Kill()

	//create client
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := http.Client{Transport: tr}
	clientAPIURL := url.URL{
		Scheme: "https",
		Host:   clientServerAddr,
		Path:   "quote",
	}

	//test get quote
	resp, err := client.Get(clientAPIURL.String())
	assert.Nil(err, err)
	assert.Equal(http.StatusOK, resp.StatusCode, "get quote failed")
	resp.Body.Close()

	//test manifest
	clientAPIURL.Path = "manifest"

	//try read before set
	buffer := make([]byte, 1024)
	resp, err = client.Get(clientAPIURL.String())
	_, readErr := resp.Body.Read(buffer)

	assert.Nil(err, err)
	assert.Equal(eof, readErr)
	assert.Contains(string(buffer), "{\"ManifestSignature\":null}")
	assert.Equal(http.StatusOK, resp.StatusCode, "status != ok")
	resp.Body.Close()

	//set Manifest
	resp, err = client.Post(clientAPIURL.String(), "application/json", bytes.NewBuffer([]byte(manifestJSON)))

	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	//read after set
	resp, err = client.Get(clientAPIURL.String())

	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.StatusCode)

	_, readErr = resp.Body.Read(buffer)
	resp.Body.Close()

	assert.Equal(eof, readErr, readErr)
	assert.NotContains(string(buffer), "{\"ManifestSignature\":null}")

	//try set manifest again
	resp, err = client.Post(clientAPIURL.String(), "application/json", bytes.NewBuffer([]byte(manifestJSON)))
	assert.Nil(err)
	assert.Equal(http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()

	//todo test status AB#121

}

type coordinatorConfig struct {
	MeshServerAddr   string
	ClientServerAddr string
	DataPath         string
}

func createCoordinatorConfig() string {
	tempDir, err := ioutil.TempDir("/tmp", "edg_coordinator_*")
	if err != nil {
		panic(err)
	}
	cfg := coordinatorConfig{MeshServerAddr: marbleServerAddr, ClientServerAddr: clientServerAddr, DataPath: tempDir}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}

	file, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}

	name := file.Name()

	_, err = file.Write(jsonCfg)
	file.Close()
	if err != nil {
		os.Remove(name)
		panic(err)
	}

	return name
}

func cleanupCoordinatorConfig(filename string) {
	jsonCfg, err := ioutil.ReadFile(filename)
	os.Remove(filename)
	if err != nil {
		panic(err)
	}
	var cfg coordinatorConfig
	if err := json.Unmarshal(jsonCfg, &cfg); err != nil {
		panic(err)
	}
	if err := os.RemoveAll(cfg.DataPath); err != nil {
		panic(err)
	}
}

func startCoordinator(configFilename string) *os.Process {
	cmd := exec.Command(*coordinatorExe, "-c", configFilename)
	output := make(chan []byte)
	go func() {
		out, _ := cmd.CombinedOutput()
		output <- out
	}()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{Transport: tr}
	url := url.URL{Scheme: "https", Host: clientServerAddr, Path: "quote"}

	log.Println("Coordinator starting ...")
	for {
		time.Sleep(10 * time.Millisecond)
		select {
		case out := <-output:
			// process died
			log.Println(string(out))
			return nil
		default:
		}
		resp, err := client.Get(url.String())
		if err == nil {
			log.Println("Coordinator started")
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				panic(resp.Status)
			}
			return cmd.Process
		}
	}
}

func setManifest(manifest core.Manifest) error {
	// Use ClientAPI to set Manifest
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{Transport: tr}
	clientAPIURL := url.URL{
		Scheme: "https",
		Host:   clientServerAddr,
		Path:   "manifest",
	}

	manifestRaw, err := json.Marshal(manifest)
	if err != nil {
		panic(err)
	}

	resp, err := client.Post(clientAPIURL.String(), "application/json", bytes.NewBuffer([]byte(manifestRaw)))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected %v, but /manifest returned %v: %v", http.StatusOK, resp.Status, string(body))
	}
	return nil
}

type marbleConfig struct {
	CoordinatorAddr string
	MarbleType      string
	DNSNames        string
	DataPath        string
}

func createMarbleConfig(coordinatorAddr, marbleType, marbleDNSNames string) string {
	cfg := marbleConfig{
		CoordinatorAddr: coordinatorAddr,
		MarbleType:      marbleType,
		DNSNames:        marbleDNSNames,
	}
	var err error
	cfg.DataPath, err = ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		os.RemoveAll(cfg.DataPath)
		panic(err)
	}

	file, err := ioutil.TempFile("", "")
	if err != nil {
		os.RemoveAll(cfg.DataPath)
		panic(err)
	}

	name := file.Name()

	_, err = file.Write(jsonCfg)
	file.Close()
	if err != nil {
		os.Remove(name)
		os.RemoveAll(cfg.DataPath)
		panic(err)
	}
	return name
}

func cleanupMarbleConfig(filename string) {
	jsonCfg, err := ioutil.ReadFile(filename)
	os.Remove(filename)
	if err != nil {
		panic(err)
	}
	var cfg marbleConfig
	if err := json.Unmarshal(jsonCfg, &cfg); err != nil {
		panic(err)
	}
	if err := os.RemoveAll(cfg.DataPath); err != nil {
		panic(err)
	}
}

func startMarble(cfgFilename string) *exec.Cmd {
	cmd := exec.Command(*marbleExe, "-c", cfgFilename)
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	log.Println("Marble started")
	return cmd
}
