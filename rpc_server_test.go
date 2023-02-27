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
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"strings"
	"testing"
	pb "tiflash-auto-scaling/supervisor_proto"
	"time"
)

var (
	addr      = "localhost:7000"
	tenantID  = "test-tenant-id"
	tenantID2 = "test-tenant-id2"
)

func InitRPCTestEnv() {
	IsTestEnv = true
	go TiFlashMaintainer()
	TiFlashBinPath = "./test_data/infinite_loop.sh"
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	defer s.Stop()
	pb.RegisterAssignServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func TestAssignAndUnassignTenant(t *testing.T) {
	go InitRPCTestEnv()
	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.FailOnNonTempDialError(true), grpc.WithBlock())
	assert.NoError(t, err)
	defer conn.Close()
	c := pb.NewAssignClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// test AssignTenantService
	assignTenantResult, err := c.AssignTenant(ctx, &pb.AssignRequest{TenantID: tenantID, TidbStatusAddr: "123.123.123.123:1000", PdAddr: "179.1.1.1:2000"})
	assert.NoError(t, err)
	assert.False(t, assignTenantResult.HasErr)
	assert.Equal(t, assignTenantResult.TenantID, tenantID)
	assert.False(t, assignTenantResult.IsUnassigning)

	assignTenantResult, err = c.AssignTenant(ctx, &pb.AssignRequest{TenantID: tenantID2, TidbStatusAddr: "123.123.123.123:1000", PdAddr: "179.1.1.1:2000"})
	assert.NoError(t, err)
	assert.True(t, assignTenantResult.HasErr)
	assert.True(t, strings.Contains(assignTenantResult.ErrInfo, "TiFlash has been occupied by a tenant"))
	assert.Equal(t, assignTenantResult.TenantID, tenantID)
	assert.False(t, assignTenantResult.IsUnassigning)

	// test GetCurrentTenantService
	getCurrentTenantResult, err := c.GetCurrentTenant(ctx, &emptypb.Empty{})
	assert.NoError(t, err)
	assert.False(t, getCurrentTenantResult.IsUnassigning)
	assert.Equal(t, getCurrentTenantResult.TenantID, tenantID)

	// test UnassignTenantService
	unassignTenantResult, err := c.UnassignTenant(ctx, &pb.UnassignRequest{AssertTenantID: tenantID2})
	assert.NoError(t, err)
	assert.True(t, unassignTenantResult.HasErr)
	assert.True(t, strings.Contains(unassignTenantResult.ErrInfo, "TiFlash is not assigned to this tenant"))
	assert.Equal(t, unassignTenantResult.TenantID, tenantID)
	assert.False(t, unassignTenantResult.IsUnassigning)

	unassignTenantResult, err = c.UnassignTenant(ctx, &pb.UnassignRequest{AssertTenantID: tenantID})
	assert.NoError(t, err)
	assert.False(t, unassignTenantResult.HasErr)
	assert.Equal(t, unassignTenantResult.TenantID, "")
	assert.False(t, unassignTenantResult.IsUnassigning)
}
