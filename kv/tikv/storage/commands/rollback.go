package commands

import (
	"github.com/pingcap-incubator/tinykv/kv/tikv/dbreader"
	"github.com/pingcap-incubator/tinykv/kv/tikv/storage/kvstore"
	"github.com/pingcap-incubator/tinykv/proto/pkg/kvrpcpb"
)

type Rollback struct {
	CommandBase
	request  *kvrpcpb.BatchRollbackRequest
	response *kvrpcpb.BatchRollbackResponse
}

func NewRollback(request *kvrpcpb.BatchRollbackRequest) Rollback {
	response := new(kvrpcpb.BatchRollbackResponse)
	return Rollback{
		CommandBase: CommandBase{
			context:  request.Context,
			response: response,
		},
		request:  request,
		response: response,
	}
}

func (r *Rollback) BuildTxn(txn *kvstore.MvccTxn) error {
	txn.StartTS = &r.request.StartVersion
	for _, k := range r.request.Keys {
		err := rollbackKey(k, txn)
		if err != nil {
			return err
		}
	}
	return nil
}

func rollbackKey(key []byte, txn *kvstore.MvccTxn) error {
	lock, err := txn.GetLock(key)
	if err != nil {
		return err
	}

	if lock == nil || lock.Ts != *txn.StartTS {
		// There is no lock, check the write status.
		existingWrite, ts, err := txn.FindWrite(key, *txn.StartTS)
		if err != nil {
			return err
		}
		if existingWrite == nil {
			// There is no write either, presumably the prewrite was lost. We insert a rollback write anyway.
			write := kvstore.Write{StartTS: *txn.StartTS, Kind: kvstore.WriteKindRollback}
			txn.PutWrite(key, &write, *txn.StartTS)

			return nil
		} else {
			if existingWrite.Kind == kvstore.WriteKindRollback {
				// The key has already been rolled back, so nothing to do.
				return nil
			}

			// The key has already been committed. This should not happen since the client should never send both
			// commit and rollback requests.
			return &Committed{key, ts}
		}
	}

	if lock.Kind == kvstore.WriteKindPut {
		txn.DeleteValue(key)
	}

	write := kvstore.Write{StartTS: *txn.StartTS, Kind: kvstore.WriteKindRollback}
	txn.PutWrite(key, &write, *txn.StartTS)
	txn.DeleteLock(key)

	return nil
}

func (r *Rollback) HandleError(err error) interface{} {
	if err == nil {
		return nil
	}

	if regionErr := extractRegionError(err); regionErr != nil {
		r.response.RegionError = regionErr
		return r.response
	}

	if e, ok := err.(KeyError); ok {
		r.response.Error = e.KeyErrors()[0]
		return r.response
	}

	return nil
}

func (r *Rollback) WillWrite(reader dbreader.DBReader) ([][]byte, error) {
	return r.request.Keys, nil
}
