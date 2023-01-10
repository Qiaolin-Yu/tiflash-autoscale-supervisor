package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	pb "tiflash-auto-scaling/supervisor_proto"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	AssignTenantID atomic.Value
	StartTime      atomic.Int64
	IsUnassigning  atomic.Bool // if IsUnassigning is true, it should be false in a few seconds
	MuOfTenantInfo sync.Mutex  // protect startTime, IsUnassigning and assignTenantID

	Pid                       atomic.Int32
	ReqId                     atomic.Int32
	MuOfSupervisor            sync.Mutex
	AssignCh                  = make(chan *pb.AssignRequest, 10000)
	LabelPatchCh              = make(chan string, 10000)
	PdAddr                    string
	TimeoutArrOfK8sLabelPatch = []int{1, 2, 4, 8, 10}
)

const NeedPd = false

var LocalPodIp string
var LocalPodName string
var K8sCli *kubernetes.Clientset

func setTenantInfo(tenantID string, isUnassigning bool) int64 {
	MuOfTenantInfo.Lock()
	defer MuOfTenantInfo.Unlock()
	AssignTenantID.Store(tenantID)
	stime := time.Now().Unix()
	StartTime.Store(stime)
	IsUnassigning.Store(isUnassigning)
	return stime
}

func getTenantInfo() (string, int64, bool) {
	MuOfTenantInfo.Lock()
	defer MuOfTenantInfo.Unlock()
	return AssignTenantID.Load().(string), StartTime.Load(), IsUnassigning.Load()
}

func AssignTenantService(req *pb.AssignRequest) (*pb.Result, error) {
	curReqId := ReqId.Add(1)
	log.Printf("received assign request by: %v reqid: %v\n", req.GetTenantID(), curReqId)
	defer log.Printf("finished assign request by: %v reqid: %v\n", req.GetTenantID(), curReqId)
	if AssignTenantID.Load().(string) == "" {
		MuOfSupervisor.Lock()
		defer MuOfSupervisor.Unlock()
		if AssignTenantID.Load().(string) == "" {
			stime := setTenantInfo(req.GetTenantID(), false)
			LabelPatchCh <- req.GetTenantID()
			configFile := fmt.Sprintf("conf/tiflash-tenant-%s.toml", req.GetTenantID())
			PdAddr = req.GetPdAddr()
			err := RenderTiFlashConf(configFile, req.GetTidbStatusAddr(), req.GetPdAddr())
			if err != nil {
				//rollback
				setTenantInfo("", false)
				LabelPatchCh <- "null"
				return &pb.Result{HasErr: true, NeedUpdateStateIfErr: true, ErrInfo: "could not render config", TenantID: "", StartTime: stime, IsUnassigning: false}, err
			}
			AssignCh <- req
			return &pb.Result{HasErr: false, ErrInfo: "", TenantID: AssignTenantID.Load().(string), StartTime: stime, IsUnassigning: false}, nil
		}
	} else if AssignTenantID.Load().(string) == req.GetTenantID() {
		realTID, stimeOfAssign, isUnassigning := getTenantInfo()
		return &pb.Result{HasErr: false, ErrInfo: "", TenantID: realTID, StartTime: stimeOfAssign, IsUnassigning: isUnassigning}, nil
	}
	realTID, stimeOfAssign, isUnassigning := getTenantInfo()
	return &pb.Result{HasErr: true, NeedUpdateStateIfErr: false, ErrInfo: "TiFlash has been occupied by a tenant", TenantID: realTID, StartTime: stimeOfAssign, IsUnassigning: isUnassigning}, nil
}

func UnassignTenantService(req *pb.UnassignRequest) (*pb.Result, error) {
	curReqId := ReqId.Add(1)
	log.Printf("received unassign request by: %v reqid: %v\n", req.GetAssertTenantID(), curReqId)
	defer log.Printf("finished unassign request by: %v reqid: %v\n", req.GetAssertTenantID(), curReqId)
	if req.AssertTenantID == AssignTenantID.Load().(string) {
		MuOfSupervisor.Lock()
		defer MuOfSupervisor.Unlock()
		if req.AssertTenantID == AssignTenantID.Load().(string) && Pid.Load() != 0 {
			// if NeedPd {
			// 	err := PdCtlNotifyPDForExit()
			// 	if err != nil {
			// 		return &pb.Result{HasErr: true, ErrInfo: err.Error(), TenantID: AssignTenantID.Load().(string)}, err
			// 	}

			// 	go func() {
			// 		log.Printf("[UnassignTenantService]RemoveStoreIDsOfUnhealthRNs \n")
			// 		err = PdCtlRemoveStoreIDsOfUnhealthRNs()
			// 		if err != nil {
			// 			log.Printf("[error]Remove StoreIDs Of Unhealth RNs fail: %v\n", err.Error())
			// 		}
			// 	}()
			// }
			if !req.ForceShutdown {
				setTenantInfo(req.AssertTenantID, true)
				log.Printf("[unassigning]wait tiflash to shutdown gracefully\n")
				time.Sleep(60 * time.Second)
			}

			setTenantInfo("", false)
			cmd := exec.Command("killall", "-9", "./bin/tiflash")
			err := cmd.Run()
			if err != nil {
				log.Printf("[error] killall tiflash failed! tenant:%v err;%v", req.AssertTenantID, err.Error())
			}
			Pid.Store(0)
			LabelPatchCh <- "null"
			// if err != nil {
			// 	return &pb.Result{HasErr: true, ErrInfo: err.Error(), TenantID: AssignTenantID.Load().(string)}, err
			// }
			return &pb.Result{HasErr: false, ErrInfo: "", TenantID: AssignTenantID.Load().(string)}, nil
		}
	}
	realTID, stimeOfAssign, isUnassigning := getTenantInfo()
	return &pb.Result{HasErr: true, NeedUpdateStateIfErr: false, ErrInfo: "TiFlash is not assigned to this tenant", TenantID: realTID, StartTime: stimeOfAssign, IsUnassigning: isUnassigning}, nil

}

func GetCurrentTenantService() (*pb.GetTenantResponse, error) {
	tenantID, startTime, isUnassigning := getTenantInfo()
	return &pb.GetTenantResponse{TenantID: tenantID, StartTime: startTime, IsUnassigning: isUnassigning}, nil
}

func K8sPodLabelPatchMaintainer() {
	for true {
		if len(LabelPatchCh) > 1 {
			log.Printf("[warning]size of patch channel > 1, size: %v\n", len(LabelPatchCh))
			for len(LabelPatchCh) > 1 {
				<-LabelPatchCh
			}
		}
		in := <-LabelPatchCh
		err := patchLabel(in)
		if err != nil {
			index := 0
			for len(LabelPatchCh) == 0 {
				time.Sleep(time.Duration(TimeoutArrOfK8sLabelPatch[index]) * time.Second)
				err = patchLabel(in)
				if err == nil {
					break
				}
				if index < len(TimeoutArrOfK8sLabelPatch)-1 {
					index++
				}
			}
		}
	}
}

func TiFlashMaintainer() {
	for true {
		if len(AssignCh) > 1 {
			log.Printf("[warning]size of assign channel > 1, size: %v\n", len(AssignCh))
			for len(AssignCh) > 1 {
				<-AssignCh
			}
		}
		in := <-AssignCh
		configFile := fmt.Sprintf("conf/tiflash-tenant-%s.toml", in.GetTenantID())
		for in.GetTenantID() == AssignTenantID.Load().(string) {
			// if NeedPd {
			// 	err := PdCtlNotifyPDForExit()
			// 	if err != nil {
			// 		log.Printf("[error]notify pd fail: %v\n", err.Error())
			// 	}
			// }
			err := os.RemoveAll("/tiflash/data")
			if err != nil {
				log.Printf("[error]remove data fail: %v\n", err.Error())
			}
			// if NeedPd {
			// 	log.Printf("[TiFlashMaintainer]RemoveStoreIDsOfUnhealthRNs \n")
			// 	err = PdCtlRemoveStoreIDsOfUnhealthRNs()
			// 	if err != nil {
			// 		log.Printf("[error]Remove StoreIDs Of Unhealth RNs fail: %v\n", err.Error())
			// 	}
			// }
			if len(AssignCh) > 0 {
				log.Printf("size of assign channel > 0, consume!\n")
				break
			}

			cmd := exec.Command("./bin/tiflash", "server", "--config-file", configFile)
			err = cmd.Start()
			Pid.Store(int32(cmd.Process.Pid))
			if err != nil {
				log.Printf("start tiflash failed: %v", err)
			}
			err = cmd.Wait()
			log.Printf("tiflash exited: %v", err)
		}
	}
}

func patchLabel(tenantId string) error {
	playLoadBytes := fmt.Sprintf(`
  {
   "metadata": {
    "labels": {
     "pod" : "%s",
     "metrics_topic" : "tiflash",
	 "pod_ip" : "%s",
     "tidb_cluster" : "%s"
    }
   }
  }
  `, LocalPodName, LocalPodIp, tenantId)

	_, err := K8sCli.CoreV1().Pods("tiflash-autoscale").Patch(context.TODO(), LocalPodName, k8stypes.StrategicMergePatchType, []byte(playLoadBytes), metav1.PatchOptions{})
	return err
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
	setTenantInfo("", false)
	LabelPatchCh <- "null"
	err = InitTiFlashConf()
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
	go TiFlashMaintainer()
	go K8sPodLabelPatchMaintainer()
}
