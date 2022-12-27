package main

import (
	"context"
	"encoding/json"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	pb "tiflash-auto-scaling/supervisor_proto"
	"time"
)

var (
	assignTenantID atomic.Value
	startTime      atomic.Int64
	muOfTenantInfo sync.Mutex // protect startTime and assignTenantID

	pid    atomic.Int32
	reqId  atomic.Int32
	mu     sync.Mutex
	ch     = make(chan *pb.AssignRequest, 10000)
	pdAddr string
)
var LocalPodIp string
var LocalPodName string
var K8sCli *kubernetes.Clientset

func setTenantInfo(tenantID string) int64 {
	muOfTenantInfo.Lock()
	defer muOfTenantInfo.Unlock()
	assignTenantID.Store(tenantID)
	stime := time.Now().Unix()
	startTime.Store(stime)
	return stime
}

func getTenantInfo() (string, int64) {
	muOfTenantInfo.Lock()
	defer muOfTenantInfo.Unlock()
	return assignTenantID.Load().(string), startTime.Load()
}

func AssignTenantService(in *pb.AssignRequest) (*pb.Result, error) {
	curReqId := reqId.Add(1)
	log.Printf("received assign request by: %v reqid: %v\n", in.GetTenantID(), curReqId)
	defer log.Printf("finished assign request by: %v reqid: %v\n", in.GetTenantID(), curReqId)
	if assignTenantID.Load().(string) == "" {
		mu.Lock()
		defer mu.Unlock()
		if assignTenantID.Load().(string) == "" {
			stime := setTenantInfo(in.GetTenantID())
			configFile := fmt.Sprintf("conf/tiflash-tenant-%s.toml", in.GetTenantID())
			pdAddr = in.GetPdAddr()
			err := RenderTiFlashConf(configFile, in.GetTidbStatusAddr(), in.GetPdAddr())
			if err != nil {
				return &pb.Result{HasErr: true, ErrInfo: "could not render config", TenantID: assignTenantID.Load().(string)}, err
			}
			ch <- in
			return &pb.Result{HasErr: false, ErrInfo: "", TenantID: assignTenantID.Load().(string), StartTime: stime}, nil
		}
	} else if assignTenantID.Load().(string) == in.GetTenantID() {
		realTID, stimeOfAssign := getTenantInfo()
		return &pb.Result{HasErr: false, ErrInfo: "", TenantID: realTID, StartTime: stimeOfAssign}, nil
	}
	return &pb.Result{HasErr: true, ErrInfo: "TiFlash has been occupied by a tenant", TenantID: assignTenantID.Load().(string)}, nil
}

func GetStoreIdsOfUnhealthRNs(str string) []string {

	var x map[string]interface{}
	err := json.Unmarshal([]byte(str), &x)
	if err != nil {
		return nil
	}
	arr, ok := x["stores"].([]interface{})
	if !ok {
		fmt.Println("#2")
		return nil
	}
	retStoreIDs := make([]string, 0, 5)
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

		rawLabelMap, ok := mapPart["labels"]
		if !ok {
			continue
		}
		rawLabels, ok := rawLabelMap.([]interface{})
		if !ok {
			continue
		}
		for _, rawLabel := range rawLabels {
			label, ok := rawLabel.(map[string]interface{})
			if !ok {
				continue
			}
			labelKey, ok := label["key"]
			if !ok {
				continue
			}
			if labelKey == "engine" {
				labelValue, ok := label["value"]
				if !ok || labelValue != "tiflash_mpp" {
					continue
				} else {
					state, ok := mapPart["state_name"]
					if ok && state != "Up" && state != "up" && state != "UP" {
						// record unhealthy RNs from PD
						sid, ok := mapPart["id"].(float64)
						if !ok {
							continue
						} else {
							retStoreIDs = append(retStoreIDs, strconv.Itoa(int(sid)))
						}
					} else {
						continue
					}
				}
			}
		}

	}
	return retStoreIDs
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
				return strconv.Itoa(int(sid))
			}
		}
	}
	return ""
}

func RemoveStoreIDsOfUnhealthRNs() error {
	outOfPdctl, err := exec.Command("./bin/pd-ctl", "-u", "http://"+pdAddr, "store").Output()
	if err != nil {
		log.Printf("[error][RemoveStoreIDsOfUnhealthRNs]pd ctl get store error: %v\n", err.Error())
		return err
	}
	sIDs := GetStoreIdsOfUnhealthRNs(string(outOfPdctl))
	if sIDs != nil {
		for _, storeID := range sIDs {
			err := RemoveStoreIDFromPD(storeID)
			if err != nil {
				continue
			}
		}
	}
	err = RemoveTombStonesFromPD()

	return err
}

func RemoveStoreIDFromPD(storeID string) error {
	if storeID != "" {
		_, err := exec.Command("./bin/pd-ctl", "-u", "http://"+pdAddr, "store", "delete", storeID).Output()
		if err != nil {
			log.Printf("[error]pd ctl get store error: %v\n", err.Error())
			return err
		}
	}
	return nil
}

func RemoveTombStonesFromPD() error {
	_, err := exec.Command("./bin/pd-ctl", "-u", "http://"+pdAddr, "store", "remove-tombstone").Output()
	if err != nil {
		log.Printf("[error]pd ctl get store error: %v\n", err.Error())
	}
	return err
}

func NotifyPDForExit() error {
	outOfPdctl, err := exec.Command("./bin/pd-ctl", "-u", "http://"+pdAddr, "store").Output()
	if err != nil {
		log.Printf("[error]pd ctl get store error: %v\n", err.Error())
		return err
	}
	storeID := FindStoreIdFromJsonStr(string(outOfPdctl))
	if storeID != "" {
		err := RemoveStoreIDFromPD(storeID)
		if err != nil {
			return err
		}
		err = RemoveTombStonesFromPD()
		if err != nil {
			return err
		}
	}
	return nil
}

func UnassignTenantService(in *pb.UnassignRequest) (*pb.Result, error) {
	curReqId := reqId.Add(1)
	log.Printf("received unassign request by: %v reqid: %v\n", in.GetAssertTenantID(), curReqId)
	defer log.Printf("finished unassign request by: %v reqid: %v\n", in.GetAssertTenantID(), curReqId)
	if in.AssertTenantID == assignTenantID.Load().(string) {
		mu.Lock()
		defer mu.Unlock()
		if in.AssertTenantID == assignTenantID.Load().(string) && pid.Load() != 0 {
			err := NotifyPDForExit()
			if err != nil {
				return &pb.Result{HasErr: true, ErrInfo: err.Error(), TenantID: assignTenantID.Load().(string)}, err
			}
			go func() {
				log.Printf("[UnassignTenantService]RemoveStoreIDsOfUnhealthRNs \n")
				err = RemoveStoreIDsOfUnhealthRNs()
				if err != nil {
					log.Printf("[error]Remove StoreIDs Of Unhealth RNs fail: %v\n", err.Error())
				}
			}()
			setTenantInfo("")
			cmd := exec.Command("killall", "-9", "./bin/tiflash")
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
	tenantID, startTime := getTenantInfo()
	return &pb.GetTenantResponse{TenantID: tenantID, StartTime: startTime}, nil
}

func TiFlashMaintainer() {
	for true {
		if len(ch) > 1 {
			log.Printf("[warning]size of channel > 1, size: %v\n", len(ch))
			for len(ch) > 1 {
				<-ch
			}
		}
		in := <-ch
		configFile := fmt.Sprintf("conf/tiflash-tenant-%s.toml", in.GetTenantID())
		for in.GetTenantID() == assignTenantID.Load().(string) {
			err := NotifyPDForExit()
			if err != nil {
				log.Printf("[error]notify pd fail: %v\n", err.Error())
			}
			err = os.RemoveAll("/tiflash/data")
			if err != nil {
				log.Printf("[error]remove data fail: %v\n", err.Error())
			}
			log.Printf("[TiFlashMaintainer]RemoveStoreIDsOfUnhealthRNs \n")
			err = RemoveStoreIDsOfUnhealthRNs()
			if err != nil {
				log.Printf("[error]Remove StoreIDs Of Unhealth RNs fail: %v\n", err.Error())
			}
			if len(ch) > 0 {
				log.Printf("size of channel > 0, consume!\n")
				break
			}

			cmd := exec.Command("./bin/tiflash", "server", "--config-file", configFile)
			err = cmd.Start()
			pid.Store(int32(cmd.Process.Pid))
			if err != nil {
				log.Printf("start tiflash failed: %v", err)
			}
			patchLabel(in.GetTenantID())
			err = cmd.Wait()
			log.Printf("tiflash exited: %v", err)
		}
	}
}

func patchLabel(tenantId string) {
	playLoadBytes := `
  {
   "metadata": {
    "labels": {
     "pod" : "` + LocalPodName + `",
     "metrics_topic" : "tiflash",
	 "pod_ip" : "` + LocalPodIp + `",
     "tidb_cluster" : "` + tenantId + `",
    }
   }
  }
  `
	_, err := K8sCli.CoreV1().Pods("default").Patch(context.TODO(), LocalPodName, k8stypes.StrategicMergePatchType, []byte(playLoadBytes), metav1.PatchOptions{})
	if err != nil {
		panic(err.Error())
	}
}

func InitService() {
	LocalPodIp = os.Getenv("POD_IP")
	LocalPodName = os.Getenv("POD_NAME")
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	K8sCli, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	setTenantInfo("")
	err = InitTiFlashConf()
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
	go TiFlashMaintainer()
}
