package tiflashrpc

import (
	"context"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/mpp"
	"time"
)

type CmdType uint16

// CmdType values.
const (
	CmdGet CmdType = 1 + iota
	CmdScan
	CmdPrewrite
	CmdCommit
	CmdCleanup
	CmdBatchGet
	CmdBatchRollback
	CmdScanLock
	CmdResolveLock
	CmdGC
	CmdDeleteRange
	CmdPessimisticLock
	CmdPessimisticRollback
	CmdTxnHeartBeat
	CmdCheckTxnStatus
	CmdCheckSecondaryLocks

	CmdRawGet CmdType = 256 + iota
	CmdRawBatchGet
	CmdRawPut
	CmdRawBatchPut
	CmdRawDelete
	CmdRawBatchDelete
	CmdRawDeleteRange
	CmdRawScan

	CmdUnsafeDestroyRange

	CmdRegisterLockObserver
	CmdCheckLockObserver
	CmdRemoveLockObserver
	CmdPhysicalScanLock

	CmdStoreSafeTS
	CmdLockWaitInfo

	CmdCop CmdType = 512 + iota
	CmdCopStream
	CmdBatchCop
	CmdMPPTask
	CmdMPPConn
	CmdMPPCancel
	CmdMPPAlive

	CmdMvccGetByKey CmdType = 1024 + iota
	CmdMvccGetByStartTs
	CmdSplitRegion

	CmdDebugGetRegionProperties CmdType = 2048 + iota

	CmdEmpty CmdType = 3072 + iota
)

type Request struct {
	Type CmdType
	Req  interface{}
	kvrpcpb.Context
	TxnScope        string
	ReplicaReadSeed *uint32 // pointer to follower read seed in snapshot/coprocessor
	StoreTp         EndpointType
	// ForwardedHost is the address of a store which will handle the request. It's different from
	// the address the request sent to.
	// If it's not empty, the store which receive the request will forward it to
	// the forwarded host. It's useful when network partition occurs.
	ForwardedHost string
}

type Response struct {
	Resp interface{}
}

type Client interface {
	// Close should release all data.
	Close() error
	// SendRequest sends Request.
	SendRequest(ctx context.Context, addr string, req *Request, timeout time.Duration) (*Response, error)
}

// IsMPPAlive returns IsAlive task in request
func (req *Request) IsMPPAlive() *mpp.IsAliveRequest {
	return req.Req.(*mpp.IsAliveRequest)
}
