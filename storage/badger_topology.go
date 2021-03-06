package storage

import (
	"encoding/binary"

	"github.com/MixinNetwork/mixin/common"
	"github.com/dgraph-io/badger"
	"github.com/vmihailenco/msgpack"
)

const snapshotsPrefixTopology = "TOPOLOGY" // local topological sorted snapshots, irreverlant to the consensus rule

func (s *BadgerStore) SnapshotsReadSnapshotsSinceTopology(topologyOffset, count uint64) ([]*common.SnapshotWithTopologicalOrder, error) {
	snapshots := make([]*common.SnapshotWithTopologicalOrder, 0)

	txn := s.snapshotsDB.NewTransaction(false)
	defer txn.Discard()

	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()

	it.Seek(topologyKey(topologyOffset))
	for ; it.ValidForPrefix([]byte(snapshotsPrefixTopology)) && uint64(len(snapshots)) < count; it.Next() {
		item := it.Item()
		v, err := item.ValueCopy(nil)
		if err != nil {
			return snapshots, err
		}
		var s common.SnapshotWithTopologicalOrder
		err = msgpack.Unmarshal(v, &s)
		if err != nil {
			return snapshots, err
		}
		s.TopologicalOrder = topologyOrder(item.Key())
		s.Hash = s.Transaction.Hash()
		snapshots = append(snapshots, &s)
	}

	return snapshots, nil
}

func (s *BadgerStore) SnapshotsTopologySequence() uint64 {
	var sequence uint64

	txn := s.snapshotsDB.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Reverse = true

	it := txn.NewIterator(opts)
	defer it.Close()

	it.Seek(topologyKey(^uint64(0)))
	if it.ValidForPrefix([]byte(snapshotsPrefixTopology)) {
		item := it.Item()
		sequence = topologyOrder(item.Key()) + 1
	}
	return sequence
}

func writeSnapshotTopology(txn *badger.Txn, s *common.SnapshotWithTopologicalOrder) error {
	key := topologyKey(s.TopologicalOrder)
	val := common.MsgpackMarshalPanic(s)
	return txn.Set(key, val)
}

func topologyKey(order uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, order)
	return append([]byte(snapshotsPrefixTopology), buf...)
}

func topologyOrder(key []byte) uint64 {
	order := key[len(snapshotsPrefixTopology):]
	return binary.BigEndian.Uint64(order)
}
