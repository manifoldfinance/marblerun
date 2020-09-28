package main

// #cgo LDFLAGS: -Wl,-unresolved-symbols=ignore-in-object-files
// void mountData(const char* path);
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/edgelesssys/coordinator/marble/cmd/common"
	"github.com/edgelesssys/coordinator/marble/marble"
)

const (
	Success             int = 0
	InternalError       int = 2
	AuthenticationError int = 4
	UsageError          int = 8
)

func main() {}

func mountData(path string) {
	C.mountData((*C.char)(unsafe.Pointer(&[]byte(path)[0])))
}

//export ert_meshentry_premain
func ert_meshentry_premain(configStr *C.char, argc *C.int, argv ***C.char) {
	config := C.GoString(configStr)

	cfg := struct {
		CoordinatorAddr string
		MarbleType      string
		DNSNames        string
		DataPath        string
	}{}
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		panic(err)
	}
	// mount data dir
	mountData(cfg.DataPath) // mounts DataPath to /marble/data
	// set env vars
	if err := os.Setenv(marble.EdgCoordinatorAddr, cfg.CoordinatorAddr); err != nil {
		log.Fatalf("failed to set env variable: %v", err)
		panic(err)
	}
	if err := os.Setenv(marble.EdgMarbleType, cfg.MarbleType); err != nil {
		log.Fatalf("failed to set env variable: %v", err)
		panic(err)
	}

	if err := os.Setenv(marble.EdgMarbleDNSNames, cfg.DNSNames); err != nil {
		log.Fatalf("failed to set env variable: %v", err)
		panic(err)
	}
	uuidFile := filepath.Join("marble", "data", "uuid")
	if err := os.Setenv(marble.EdgMarbleUUIDFile, uuidFile); err != nil {
		log.Fatalf("failed to set env variable: %v", err)
		panic(err)
	}

	// call PreMain
	err := marble.PreMain()
	if err != nil {
		panic(err)
	}
	ret := common.PremainTarget(len(os.Args), os.Args, os.Environ())
	if ret != 0 {
		panic(fmt.Errorf("premainTarget returned: %v", ret))
	}
	log.Println("Successfully authenticated with Coordinator!")
}
