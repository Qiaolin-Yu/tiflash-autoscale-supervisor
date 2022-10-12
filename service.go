package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	pb "tiflash-auto-scaling/supervisor_proto"
)

var (
	assignTenantID atomic.Value
	pid            atomic.Int32
	mu             sync.Mutex
	ch             = make(chan *pb.AssignRequest)
	pdAddr         string
)
var LocalPodIp string

func AssignTenantService(in *pb.AssignRequest) (*pb.Result, error) {
	log.Printf("received assign request by: %v", in.GetTenantID())
	if assignTenantID.Load().(string) == "" {
		mu.Lock()
		defer mu.Unlock()
		if assignTenantID.Load().(string) == "" {
			assignTenantID.Store(in.GetTenantID())
			configFile := fmt.Sprintf("conf/tiflash-tenant-%s.toml", in.GetTenantID())
			pdAddr = in.GetPdAddr()
			err := RenderTiFlashConf(configFile, in.GetTidbStatusAddr(), in.GetPdAddr())
			if err != nil {
				return &pb.Result{HasErr: true, ErrInfo: "could not render config", TenantID: assignTenantID.Load().(string)}, err
			}
			ch <- in
			return &pb.Result{HasErr: false, ErrInfo: "", TenantID: assignTenantID.Load().(string)}, nil
		}
	} else if assignTenantID.Load().(string) == in.GetTenantID() {
		return &pb.Result{HasErr: false, ErrInfo: "", TenantID: assignTenantID.Load().(string)}, nil
	}
	return &pb.Result{HasErr: true, ErrInfo: "TiFlash has been occupied by a tenant", TenantID: assignTenantID.Load().(string)}, nil
}

func FindStoreIdFromJsonStr(str string) string {
	var x map[string]interface{}
	err := json.Unmarshal([]byte(str), &x)
	if err != nil {
		return ""
	}
	arr, ok := x["stores"].([]interface{})
	if !ok {
		return ""
	}
	for _, item := range arr {
		jsonmap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		storeMap, ok := jsonmap["store"]
		if !ok {
			continue
		}
		mapPart, ok := storeMap.(map[string]interface{})
		if !ok {
			continue
		}
		// fmt.Println(mapPart)
		storeAddr, ok := mapPart["address"]
		if !ok {
			continue
		}
		storeAddrStr, ok := storeAddr.(string)
		if !ok {
			continue
		}
		addrArr := strings.Split(storeAddrStr, ":")
		if len(addrArr) == 0 {
			continue
		}
		if addrArr[0] == LocalPodIp {
			sid, ok := mapPart["id"].(float64)
			if !ok {
				return ""
			} else {
				return string(int(sid))
			}
		}
	}
	return ""
}

func UnassignTenantService(in *pb.UnassignRequest) (*pb.Result, error) {
	log.Printf("received unassign request by: %v", in.GetAssertTenantID())
	if in.AssertTenantID == assignTenantID.Load().(string) {
		mu.Lock()
		defer mu.Unlock()
		if in.AssertTenantID == assignTenantID.Load().(string) && pid.Load() != 0 {
			outOfPdctl, err := exec.Command("./bin/pd-ctl -u http://" + pdAddr + " store").Output()
			if err != nil {
				log.Printf("[error]pd ctl get store error: %v\n", err.Error())
			} else {
				storeID := FindStoreIdFromJsonStr(string(outOfPdctl))
				if storeID != "" {
					_, err = exec.Command("./bin/pd-ctl -u http://" + pdAddr + " store delete " + storeID).Output()
					if err != nil {
						log.Printf("[error]pd ctl get store error: %v\n", err.Error())
					} else {
						_, err = exec.Command("./bin/pd-ctl -u http://" + pdAddr + " store remove-tombstone ").Output()
						if err != nil {
							log.Printf("[error]pd ctl get store error: %v\n", err.Error())
						}
					}
				}
			}
			assignTenantID.Store("")
			cmd := exec.Command("kill", "-9", fmt.Sprintf("%v", pid.Load()))
			err = cmd.Run()
			pid.Store(0)
			if err != nil {
				return &pb.Result{HasErr: true, ErrInfo: err.Error(), TenantID: assignTenantID.Load().(string)}, err
			}
			return &pb.Result{HasErr: false, ErrInfo: "", TenantID: assignTenantID.Load().(string)}, nil
		}
	}
	return &pb.Result{HasErr: true, ErrInfo: "TiFlash is not assigned to this tenant", TenantID: assignTenantID.Load().(string)}, nil

}

func GetCurrentTenantService() (*pb.GetTenantResponse, error) {
	return &pb.GetTenantResponse{TenantID: assignTenantID.Load().(string)}, nil
}

func TiFlashMaintainer() {
	for true {
		in := <-ch
		configFile := fmt.Sprintf("conf/tiflash-tenant-%s.toml", in.GetTenantID())
		for in.GetTenantID() == assignTenantID.Load().(string) {
			err := os.RemoveAll("/tiflash/data")
			if err != nil {
				log.Printf("[error]remove data fail: %v\n", err.Error())
			}

			cmd := exec.Command("./bin/tiflash", "server", "--config-file", configFile)
			err = cmd.Start()
			pid.Store(int32(cmd.Process.Pid))
			if err != nil {
				log.Printf("start tiflash failed: %v", err)
			}
			err = cmd.Wait()
			log.Printf("tiflash exited: %v", err)
		}
	}
}

func InitService() {
	assignTenantID.Store("")
	err := InitTiFlashConf()
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
	go TiFlashMaintainer()
}
