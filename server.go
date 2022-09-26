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
	pb "tiflash-auto-scaling/rpc"
)

var (
	port = flag.Int("port", 7000, "The server port")
)

type server struct {
	pb.UnimplementedAssignServer
}

func (s *server) AssignTenant(ctx context.Context, in *pb.AssignRequest) (*pb.Result, error) {
	log.Printf("received assign request by: %v", in.GetTenantID())
	configFile := fmt.Sprintf("tiflash-%s.toml", in.GetTenantID())
	f, err := os.Create(configFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	_, err2 := f.WriteString(in.GetTenantConfig())

	if err2 != nil {
		log.Fatal(err2)
	}
	StartTiFlash(configFile)
	return &pb.Result{HasErr: false, ErrInfo: ""}, nil
}

func (s *server) UnassignTenant(ctx context.Context, in *pb.UnassignRequest) (*pb.Result, error) {
	log.Printf("received unassign request by: %v", in.GetAssertTenantID())
	return &pb.Result{HasErr: false, ErrInfo: ""}, nil
}

func StartTiFlash(configFile string) {
	cmd := exec.Command("./tiflash", "server", "--config-file", configFile)
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	flag.Parse()
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
