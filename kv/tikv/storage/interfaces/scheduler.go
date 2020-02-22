package interfaces

import (
	"github.com/pingcap-incubator/tinykv/kv/tikv/dbreader"
	"github.com/pingcap-incubator/tinykv/kv/tikv/storage/kvstore"
	"github.com/pingcap-incubator/tinykv/proto/pkg/kvrpcpb"
)

// Interfaces used with the scheduler and related code in storage.

// Scheduler takes Commands and runs them asynchronously. It is up to implementations to decide the scheduling policy.
type Scheduler interface {
	// Run executes a command asynchronously and returns a channel over which the result of the command execution is
	// sent.
	Run(Command) <-chan SchedResult
	Stop()
}

// Command is an abstraction which covers the process from receiving a request from gRPC to returning a response.
// That process is driven by a Scheduler.
type Command interface {
	// Execute is for building writes mvcc transaction. Commands can also make non-transactional
	// reads and writes using txn. Returning without modifying txn means that no transaction will be executed.
	Execute(txn *kvstore.MvccTxn) (interface{}, error)
	Context() *kvrpcpb.Context
	// Response builds a success response to return to the client.
	// Response() interface{}
	// HandleError gives the command an opportunity to handle errors generated at any stage in processing. If the command
	// handles the error, it should return a success response to return to the user. If nil is returned then an error
	// will be generated by the scheduler. Note that there is no way to continue processing after an error, even if the
	// command handles the error.
	// HandleError(err error) interface{}
	// WillWrite returns a list of all keys that might be written by this command.
	WillWrite(reader dbreader.DBReader) ([][]byte, error)
}

// SchedResult is a 'generic' result type for responses. It is used to return a Response/error pair over channels where
// we can't use Go's multiple return values.
type SchedResult struct {
	Response interface{}
	Err      error
}

func RespOk(resp interface{}) SchedResult {
	return SchedResult{
		Response: resp,
		Err:      nil,
	}
}

func RespErr(err error) SchedResult {
	return SchedResult{
		Response: nil,
		Err:      err,
	}
}
