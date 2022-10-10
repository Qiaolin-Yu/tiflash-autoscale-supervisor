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
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "tiflash-auto-scaling/supervisor_proto"
)

var (
	addr             = "localhost:7000"
	tenantID         = "demo123821"
	tenantConfigFile = "conf/tiflash-templete.toml"
)

func TestAssignTenant(t *testing.T) {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewAssignClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.AssignTenant(ctx, &pb.AssignRequest{TenantID: tenantID, TidbStatusAddr: "123.123.123.123", PdAddr: "125.125.125.125"})
	if err != nil {
		log.Fatalf("could not assign: %v", err)
	}
	if r.HasErr {
		log.Fatalf("assign failed: %v", r.ErrInfo)
	} else {
		log.Printf("assign succeeded")
	}

}

func TestGetCurrentTenant(t *testing.T) {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewAssignClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.GetCurrentTenant(ctx, &emptypb.Empty{})
	log.Printf("current tenant: %v", r.TenantID)
}

func TestUnassignTenant(t *testing.T) {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewAssignClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.UnassignTenant(ctx, &pb.UnassignRequest{AssertTenantID: tenantID})
	if err != nil {
		log.Fatalf("could not unassign: %v", err)
	}
	if r.HasErr {
		log.Fatalf("unassign failed: %v", r.ErrInfo)
	} else {
		log.Printf("unassign succeeded")
	}
}

func TestInitTiFlashConf(t *testing.T) {
	err := InitTiFlashConf()
	if err != nil {
		log.Fatalf("init tiflash conf failed: %v", err)
	}
	RenderTiFlashConf("conf/tiflash.toml", "123.123.123.123", "125.125.125.125")
}
