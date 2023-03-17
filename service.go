package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	pb "tiflash-auto-scaling/supervisor_proto"
	"time"

	"github.com/prometheus/common/expfmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
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

	TiFlashMetricURL = "http://127.0.0.1:8234/metrics"
	TiFlashBinPath   = "./bin/tiflash"
	IsTestEnv        = false

	PathOfTiflashData      = "/tiflash/data"
	PathOfTiflashCache     = "/tiflash/cache"
	CapicityOfTiflashCache = "10737418240"
)

const NeedPd = false

var S3BucketForTiFLashLog = ""
var S3Mutex sync.Mutex
var LocalPodIp string
var LocalPodName string
var CheckTiFlashIdleInitSleepSec = 10
var CheckTiFlashIdleInterval = 1
var CheckTiFlashIdleTimeout = 60
var K8sCli *kubernetes.Clientset

const CheckTiflashIdleTimeoutEnv = "CHECK_TIFLASH_IDLE_TIMEOUT"
const CheckTiflashIdleIntervalEnv = "CHECK_TIFLASH_IDLE_INTERVAL"
const TiFlashMetricTaskPrefix = "tiflash_coprocessor_handling_request_count"
const HTTPTimeout = 5 * time.Second

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

func GetTiFlashTaskNum() (int, error) {
	client := http.Client{
		Timeout: HTTPTimeout,
	}
	resp, err := client.Get(TiFlashMetricURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, errors.New("http status code is not 200")
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	res, err := GetTiFlashTaskNumByMetricsByte(data)
	if err != nil {
		return 0, err
	}
	return res, nil
}

func GetTiFlashTaskNumByMetricsByte(data []byte) (int, error) {
	res := 0
	reader := bytes.NewReader(data)
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return 0, err
	}
	for _, v := range metricFamilies {
		if strings.HasPrefix(*v.Name, TiFlashMetricTaskPrefix) {
			for _, m := range v.Metric {
				res += int(*m.Gauge.Value)
			}
			break
		}

	}
	return res, nil
}

const TiFlashListenTcpPort = "9000"

func isTiflashPortOpen() bool {
	host := "localhost"
	port := TiFlashListenTcpPort
	timeout := 100 * time.Millisecond
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	if conn != nil {
		defer conn.Close()
		return true
	} else {
		return false
	}
}

func AssignTenantService(req *pb.AssignRequest) (*pb.Result, error) {
	curReqId := ReqId.Add(1)
	log.Printf("received assign request by: %v reqid: %v\n", req.GetTenantID(), curReqId)
	var errInfo string
	defer log.Printf("finished assign request by: %v reqid: %v\n", req.GetTenantID(), curReqId)
	if AssignTenantID.Load() == nil || AssignTenantID.Load().(string) == "" {
		if MuOfSupervisor.TryLock() {
			defer MuOfSupervisor.Unlock()
			if AssignTenantID.Load() == nil || AssignTenantID.Load().(string) == "" {
				stime := setTenantInfo(req.GetTenantID(), false)
				LabelPatchCh <- req.GetTenantID()
				configFile := fmt.Sprintf("conf/tiflash-tenant-%s.toml", req.GetTenantID())
				PdAddr = req.GetPdAddr()
				err := RenderTiFlashConf(configFile, req.GetTidbStatusAddr(), req.GetPdAddr(), req.GetTenantID())
				if err != nil {
					//rollback
					setTenantInfo("", false)
					LabelPatchCh <- "null"
					return &pb.Result{HasErr: true, NeedUpdateStateIfErr: true, ErrInfo: "could not render config", TenantID: "", StartTime: stime, IsUnassigning: false}, err
				}
				AssignCh <- req
				for Pid.Load() == 0 {
					time.Sleep(10 * time.Microsecond)
				}
				localSt := time.Now()
				MaxWaitPortOpenTimeSec := 5.0
				isTimeout := false
				for !isTiflashPortOpen() && !IsTestEnv {
					time.Sleep(100 * time.Microsecond)
					if time.Since(localSt).Seconds() >= MaxWaitPortOpenTimeSec {
						isTimeout = true
						break
					}
				}
				if isTimeout {
					log.Printf("wait tiflash port open timeout! %vs\n", MaxWaitPortOpenTimeSec)
				}
				return &pb.Result{HasErr: false, ErrInfo: "", TenantID: AssignTenantID.Load().(string), StartTime: stime, IsUnassigning: false}, nil
			}
		} else {
			errInfo = "TryLock failed"
		}
	} else if AssignTenantID.Load().(string) == req.GetTenantID() {
		realTID, stimeOfAssign, isUnassigning := getTenantInfo()
		return &pb.Result{HasErr: false, ErrInfo: "", TenantID: realTID, StartTime: stimeOfAssign, IsUnassigning: isUnassigning}, nil
	} else {
		errInfo = "TiFlash has been occupied by a tenant"
	}
	realTID, stimeOfAssign, isUnassigning := getTenantInfo()
	log.Printf("[error][assign]%v realTID:%v wantTID:%v\n", errInfo, realTID, req.TenantID)
	return &pb.Result{HasErr: true, NeedUpdateStateIfErr: true, ErrInfo: "TiFlash has been occupied by a tenant", TenantID: realTID, StartTime: stimeOfAssign, IsUnassigning: isUnassigning}, nil
}

func UnassignTenantService(req *pb.UnassignRequest) (*pb.Result, error) {
	curReqId := ReqId.Add(1)
	log.Printf("received unassign request by: %v reqid: %v\n", req.GetAssertTenantID(), curReqId)
	defer log.Printf("finished unassign request by: %v reqid: %v\n", req.GetAssertTenantID(), curReqId)
	var errInfo string
	if req.AssertTenantID == AssignTenantID.Load().(string) {
		if MuOfSupervisor.TryLock() {
			defer MuOfSupervisor.Unlock()
			if req.AssertTenantID == AssignTenantID.Load().(string) && Pid.Load() != 0 {
				TryToUploadTiFlashLogIntoS3(false)
				if NeedPd {
					err := PdCtlNotifyPDForExit()
					if err != nil {
						return &pb.Result{HasErr: true, ErrInfo: err.Error(), TenantID: AssignTenantID.Load().(string)}, err
					}

					go func() {
						log.Printf("[UnassignTenantService]RemoveStoreIDsOfUnhealthRNs \n")
						err = PdCtlRemoveStoreIDsOfUnhealthRNs()
						if err != nil {
							log.Printf("[error]Remove StoreIDs Of Unhealth RNs fail: %v\n", err.Error())
						}
					}()
				}
				if !req.ForceShutdown {
					setTenantInfo(req.AssertTenantID, true)
					log.Printf("[unassigning]wait tiflash to shutdown gracefully\n")
					startTime := time.Now()
					time.Sleep(time.Duration(CheckTiFlashIdleInitSleepSec) * time.Second) // sleep a few seconds to prevent new mpp tasks arrive shortly after the begining of unassigning,  but  the tiflash is idle at the begining of unassigning
					for time.Now().Sub(startTime).Seconds() < float64(CheckTiFlashIdleTimeout) {
						if IsTestEnv {
							log.Printf("[test][unassigning]tiflash has no task, shutdown\n")
							break
						}
						taskNum, err := GetTiFlashTaskNum()
						if err != nil {
							log.Printf("[error]GetTiFlashTaskNum fail: %v\n", err.Error())
							break
						}
						if taskNum == 0 {
							log.Printf("[unassigning]tiflash has no task, shutdown\n")
							break
						}
						log.Printf("[unassigning]tiflash has %v task, wait for %v seconds\n", taskNum, CheckTiFlashIdleTimeout)
						time.Sleep(time.Duration(CheckTiFlashIdleInterval) * time.Second)
					}
				}

				setTenantInfo("", false)
				cmd := exec.Command("killall", "-9", TiFlashBinPath)
				err := cmd.Run()
				if err != nil {
					log.Printf("[error] killall tiflash failed! tenant:%v err;%v", req.AssertTenantID, err.Error())
				}
				Pid.Store(0)
				LabelPatchCh <- "null"
				TryToUploadTiFlashLogIntoS3(true)
				// if err != nil {
				// 	return &pb.Result{HasErr: true, ErrInfo: err.Error(), TenantID: AssignTenantID.Load().(string)}, err
				// }
				return &pb.Result{HasErr: false, ErrInfo: "", TenantID: AssignTenantID.Load().(string)}, nil
			}
		} else {
			errInfo = "TryLock failed"
		}
	} else {
		errInfo = "TiFlash is not assigned to this tenant"
	}
	realTID, stimeOfAssign, isUnassigning := getTenantInfo()
	log.Printf("[error][unassign]%v realTID:%v wantTID:%v\n", errInfo, realTID, req.AssertTenantID)
	return &pb.Result{HasErr: true, NeedUpdateStateIfErr: false, ErrInfo: errInfo, TenantID: realTID, StartTime: stimeOfAssign, IsUnassigning: isUnassigning}, nil

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
		err := os.RemoveAll(PathOfTiflashCache)
		if err != nil {
			log.Printf("[error]remove cache fail: %v\n", err.Error())
		}
		err = os.RemoveAll(PathOfTiflashData)
		if err != nil {
			log.Printf("[error]remove data fail: %v\n", err.Error())
		}
		for in.GetTenantID() == AssignTenantID.Load().(string) {
			if NeedPd {
				err := PdCtlNotifyPDForExit()
				if err != nil {
					log.Printf("[error]notify pd fail: %v\n", err.Error())
				}
			}
			if NeedPd {
				log.Printf("[TiFlashMaintainer]RemoveStoreIDsOfUnhealthRNs \n")
				err = PdCtlRemoveStoreIDsOfUnhealthRNs()
				if err != nil {
					log.Printf("[error]Remove StoreIDs Of Unhealth RNs fail: %v\n", err.Error())
				}
			}
			if len(AssignCh) > 0 {
				log.Printf("size of assign channel > 0, consume!\n")
				break
			}

			cmd := exec.Command(TiFlashBinPath, "server", "--config-file", configFile)
			err = cmd.Start()
			if err != nil {
				log.Printf("[error]Start TiFlash failed: %v", err)
			}
			Pid.Store(int32(cmd.Process.Pid))
			err = cmd.Wait()
			if err != nil {
				log.Printf("[error]TiFlash exited with error: %v", err)
			} else {
				log.Printf("TiFlash exited successfully")
			}
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

func outsideConfig() (*rest.Config, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	return clientcmd.BuildConfigFromFlags("", *kubeconfig)
}

func getK8sConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return outsideConfig()
	} else {
		return config, err
	}
}

func TryToUploadTiFlashLogIntoS3(isSync bool) {
	if S3BucketForTiFLashLog != "" {
		fn := func() {
			if S3Mutex.TryLock() {
				defer S3Mutex.Unlock()
				_, err := exec.Command("aws", "s3", "cp", "--recursive", "/tiflash/log/", fmt.Sprintf("s3://%v/%v/", S3BucketForTiFLashLog, LocalPodName)).Output()
				if err != nil {
					log.Printf("[error][s3] err:%v\n", err.Error())
				}
			}
		}
		if isSync {
			fn()
		} else {
			go fn()
		}
	}
}

func InitService() {
	LocalPodIp = os.Getenv("POD_IP")
	LocalPodName = os.Getenv("POD_NAME")
	S3BucketForTiFLashLog = os.Getenv("S3_FOR_TIFLASH_LOG")
	CheckTiFlashIdleTimeoutString := os.Getenv(CheckTiflashIdleTimeoutEnv)
	envtiflashCachePath := os.Getenv("TIFLASH_CACHE_PATH")
	envtiflashCacheCap := os.Getenv("TIFLASH_CACHE_CAP")
	if envtiflashCachePath != "" {
		PathOfTiflashCache = envtiflashCachePath
	}
	if envtiflashCacheCap != "" {
		CapicityOfTiflashCache = envtiflashCacheCap
	}
	var err error
	if CheckTiFlashIdleTimeoutString != "" {
		CheckTiFlashIdleTimeout, err = strconv.Atoi(CheckTiFlashIdleTimeoutString)
		if err != nil {
			panic(err.Error())
		}
	}
	CheckTiFlashIdleIntervalString := os.Getenv(CheckTiflashIdleIntervalEnv)
	if CheckTiFlashIdleIntervalString != "" {
		CheckTiFlashIdleInterval, err = strconv.Atoi(CheckTiFlashIdleIntervalString)
		if err != nil {
			panic(err.Error())
		}
	}

	config, err := getK8sConfig()
	if err != nil {
		panic(err.Error())
	}
	K8sCli, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	setTenantInfo("", false)
	LabelPatchCh <- "null"
	err = InitTiFlashConf(LocalPodIp)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
	go TiFlashMaintainer()
	go K8sPodLabelPatchMaintainer()
}
