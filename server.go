/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	pb "tiflash-auto-scaling/rpc"
)

var (
	port = flag.Int("port", 7000, "The server port")
)

type server struct {
	pb.UnimplementedAssignServer
}

var assignTenantID atomic.Value
var pid atomic.Int32
var mu sync.Mutex
var ch = make(chan *pb.AssignRequest)

func (s *server) AssignTenant(ctx context.Context, in *pb.AssignRequest) (*pb.Result, error) {
	log.Printf("received assign request by: %v", in.GetTenantID())
	if assignTenantID.Load().(string) == "" {
		mu.Lock()
		defer mu.Unlock()
		if assignTenantID.Load().(string) == "" {
			assignTenantID.Store(in.GetTenantID())
			ch <- in
			return &pb.Result{HasErr: false, ErrInfo: ""}, nil
		}
	} else if assignTenantID.Load().(string) == in.GetTenantID() {
		return &pb.Result{HasErr: false, ErrInfo: ""}, nil
	}
	return &pb.Result{HasErr: true, ErrInfo: "TiFlash has been occupied by a tenant"}, nil
}

func (s *server) UnassignTenant(ctx context.Context, in *pb.UnassignRequest) (*pb.Result, error) {
	log.Printf("received unassign request by: %v", in.GetAssertTenantID())
	if in.AssertTenantID == assignTenantID.Load() {
		mu.Lock()
		defer mu.Unlock()
		if in.AssertTenantID == assignTenantID.Load() && pid.Load() != 0 {
			assignTenantID.Store("")
			cmd := exec.Command("kill", "-9", fmt.Sprintf("%v", pid.Load()))
			err := cmd.Run()
			pid.Store(0)
			if err != nil {
				return &pb.Result{HasErr: true, ErrInfo: err.Error()}, err
			}
			return &pb.Result{HasErr: false, ErrInfo: ""}, nil
		}
	}
	return &pb.Result{HasErr: true, ErrInfo: "TiFlash is not assigned to this tenant"}, nil

}

func consumer() {
	for true {
		in := <-ch
		configFile := fmt.Sprintf("tiflash-%s.toml", in.GetTenantID())
		f, err := os.Create(configFile)
		if err != nil {
			log.Fatalf("create config file failed: %v", err)
		}
		defer f.Close()
		_, err = f.WriteString(in.GetTenantConfig())

		if err != nil {
			log.Fatalf("write config file failed: %v", err)
		}

		for assignTenantID.Load().(string) == in.GetTenantID() {
			cmd := exec.Command("./tiflash", "server", "--config-file", configFile)
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

func main() {
	flag.Parse()
	assignTenantID.Store("")
	pid.Store(0)
	go consumer()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterAssignServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
