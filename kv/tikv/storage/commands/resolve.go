package commands

import (
	"github.com/pingcap-incubator/tinykv/kv/tikv/dbreader"
	"github.com/pingcap-incubator/tinykv/kv/tikv/storage/kvstore"
	"github.com/pingcap-incubator/tinykv/proto/pkg/kvrpcpb"
)

type ResolveLock struct {
	CommandBase
	request  *kvrpcpb.ResolveLockRequest
	keyLocks []kvstore.KlPair
}

func NewResolveLock(request *kvrpcpb.ResolveLockRequest) ResolveLock {
	return ResolveLock{
		CommandBase: CommandBase{
			context: request.Context,
		},
		request: request,
	}
}

func (rl *ResolveLock) Execute(txn *kvstore.MvccTxn) (interface{}, error) {
	// A map from start timestamps to commit timestamps which tells us whether a transaction (identified by start ts)
	// has been committed (and if so, then its commit ts) or rolled back (in which case the commit ts is 0).
	txn.StartTS = &rl.request.StartVersion
	commitTs := rl.request.CommitVersion
	response := new(kvrpcpb.ResolveLockResponse)

	for _, kl := range rl.keyLocks {
		if commitTs == 0 {
			resp, err := rollbackKey(kl.Key, txn, response)
			if resp != nil || err != nil {
				return resp, err
			}
		} else {
			resp, err := commitKey(kl.Key, commitTs, txn, response)
			if resp != nil || err != nil {
				return resp, err
			}
		}
	}

	return response, nil
}

func (rl *ResolveLock) WillWrite(reader dbreader.DBReader) ([][]byte, error) {
	// Find all locks where the lock's transaction (start ts) is in txnStatus.
	keyLocks, err := kvstore.AllLocksForTxn(rl.request.StartVersion, reader)
	if err != nil {
		return nil, err
	}
	rl.keyLocks = keyLocks
	var keys [][]byte
	for _, kl := range keyLocks {
		keys = append(keys, kl.Key)
	}
	return keys, nil
}
