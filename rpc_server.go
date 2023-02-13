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
	"log"
	"net"
	pb "tiflash-auto-scaling/supervisor_proto"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	port = flag.Int("port", 7000, "The server port")
)

type server struct {
	// pb.UnimplementedAssignServer
}

func (s *server) AssignTenant(ctx context.Context, in *pb.AssignRequest) (*pb.Result, error) {
	return AssignTenantService(in)
}

func (s *server) UnassignTenant(ctx context.Context, in *pb.UnassignRequest) (*pb.Result, error) {
	return UnassignTenantService(in)
}

func (s *server) GetCurrentTenant(ctx context.Context, empty *emptypb.Empty) (*pb.GetTenantResponse, error) {
	return GetCurrentTenantService()
}

func main() {
	// LocalPodIp = os.Getenv("POD_IP")
	log.Printf("ver: 2")
	flag.Parse()
	InitService()
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
