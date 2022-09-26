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
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "tiflash-auto-scaling/rpc"
)

const (
	defaultTenantID     = "weu823u"
	defaultTenantConfig = "tiflash.toml"
)

var (
	addr         = flag.String("addr", "localhost:7000", "the address to connect to")
	tenantID     = flag.String("tenantID", defaultTenantID, "the id of tenant")
	tenantConfig = flag.String("tenantConfig", defaultTenantConfig, "the config of tenant")
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewAssignClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.AssignTenant(ctx, &pb.AssignRequest{TenantID: *tenantID, TenantConfig: *tenantConfig})
	if err != nil {
		log.Fatalf("could not assign: %v", err)
	}
	if r.HasErr {
		log.Fatalf("assign failed: %v", r.ErrInfo)
	} else {
		log.Printf("assign succeeded")
	}

	//r, err = c.UnassignTenant(ctx, &pb.UnassignRequest{AssertTenantID: *tenantID})
	//if err != nil {
	//	log.Fatalf("could not unassign: %v", err)
	//}
	//if r.HasErr {
	//	log.Fatalf("unassign failed: %v", r.ErrInfo)
	//} else {
	//	log.Printf("unassign succeeded")
	//}
}
