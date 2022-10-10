package main

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"sync/atomic"
	pb "tiflash-auto-scaling/supervisor_proto"
)

var (
	assignTenantID atomic.Value
	pid            atomic.Int32
	mu             sync.Mutex
	ch             = make(chan *pb.AssignRequest)
)

func AssignTenantService(in *pb.AssignRequest) (*pb.Result, error) {
	log.Printf("received assign request by: %v", in.GetTenantID())
	if assignTenantID.Load().(string) == "" {
		mu.Lock()
		defer mu.Unlock()
		if assignTenantID.Load().(string) == "" {
			assignTenantID.Store(in.GetTenantID())
			configFile := fmt.Sprintf("conf/tiflash-tenant-%s.toml", in.GetTenantID())
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

func UnassignTenantService(in *pb.UnassignRequest) (*pb.Result, error) {
	log.Printf("received unassign request by: %v", in.GetAssertTenantID())
	if in.AssertTenantID == assignTenantID.Load().(string) {
		mu.Lock()
		defer mu.Unlock()
		if in.AssertTenantID == assignTenantID.Load().(string) && pid.Load() != 0 {
			assignTenantID.Store("")
			cmd := exec.Command("kill", "-9", fmt.Sprintf("%v", pid.Load()))
			err := cmd.Run()
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
	mu.Lock()
	defer mu.Unlock()
	return &pb.GetTenantResponse{TenantID: assignTenantID.Load().(string)}, nil
}

func TiFlashMaintainer() {
	for true {
		in := <-ch
		configFile := fmt.Sprintf("conf/tiflash-tenant-%s.toml", in.GetTenantID())
		for in.GetTenantID() == assignTenantID.Load().(string) {
			cmd := exec.Command("./bin/tiflash", "server", "--config-file", configFile)
			err := cmd.Start()
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
