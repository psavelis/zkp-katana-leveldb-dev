package storage

import (
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// LevelDBStore wraps goleveldb for persistent key-value storage.
type LevelDBStore struct {
	db   *leveldb.DB
	path string
}

// NewLevelDBStore opens or creates a LevelDB database at the given path.
func NewLevelDBStore(path string) (*LevelDBStore, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb at %s: %w", path, err)
	}

	return &LevelDBStore{
		db:   db,
		path: path,
	}, nil
}

// NewLevelDBStoreWithOptions opens a LevelDB with custom options.
func NewLevelDBStoreWithOptions(path string, opts *opt.Options) (*LevelDBStore, error) {
	db, err := leveldb.OpenFile(path, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb at %s: %w", path, err)
	}

	return &LevelDBStore{
		db:   db,
		path: path,
	}, nil
}

// Put stores a key-value pair.
func (s *LevelDBStore) Put(key, value []byte) error {
	return s.db.Put(key, value, nil)
}

// PutWithOptions stores a key-value pair with write options.
func (s *LevelDBStore) PutWithOptions(key, value []byte, opts *opt.WriteOptions) error {
	return s.db.Put(key, value, opts)
}

// Get retrieves a value by key.
func (s *LevelDBStore) Get(key []byte) ([]byte, error) {
	value, err := s.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return nil, ErrNotFound
	}
	return value, err
}

// Delete removes a key-value pair.
func (s *LevelDBStore) Delete(key []byte) error {
	return s.db.Delete(key, nil)
}

// Has checks if a key exists.
func (s *LevelDBStore) Has(key []byte) (bool, error) {
	return s.db.Has(key, nil)
}

// BatchWrite performs multiple operations atomically.
func (s *LevelDBStore) BatchWrite(ops []BatchOp) error {
	batch := new(leveldb.Batch)
	for _, op := range ops {
		if op.Delete {
			batch.Delete(op.Key)
		} else {
			batch.Put(op.Key, op.Value)
		}
	}
	return s.db.Write(batch, nil)
}

// NewIterator creates an iterator over a key range.
func (s *LevelDBStore) NewIterator(prefix []byte) iterator.Iterator {
	return s.db.NewIterator(util.BytesPrefix(prefix), nil)
}

// NewRangeIterator creates an iterator for a specific range.
func (s *LevelDBStore) NewRangeIterator(start, limit []byte) iterator.Iterator {
	return s.db.NewIterator(&util.Range{Start: start, Limit: limit}, nil)
}

// Close closes the database.
func (s *LevelDBStore) Close() error {
	return s.db.Close()
}

// Path returns the database path.
func (s *LevelDBStore) Path() string {
	return s.path
}

// CompactRange compacts the database for the given key range.
func (s *LevelDBStore) CompactRange(start, limit []byte) error {
	return s.db.CompactRange(util.Range{Start: start, Limit: limit})
}

// Stats returns database statistics.
func (s *LevelDBStore) Stats() (*leveldb.DBStats, error) {
	var stats leveldb.DBStats
	err := s.db.Stats(&stats)
	return &stats, err
}

// GetProperty returns a database property.
func (s *LevelDBStore) GetProperty(name string) (string, error) {
	return s.db.GetProperty(name)
}

// ForEach iterates over all keys with the given prefix.
func (s *LevelDBStore) ForEach(prefix []byte, fn func(key, value []byte) error) error {
	iter := s.NewIterator(prefix)
	defer iter.Release()

	for iter.Next() {
		if err := fn(iter.Key(), iter.Value()); err != nil {
			return err
		}
	}

	return iter.Error()
}

// Count counts keys with the given prefix.
func (s *LevelDBStore) Count(prefix []byte) (int, error) {
	iter := s.NewIterator(prefix)
	defer iter.Release()

	count := 0
	for iter.Next() {
		count++
	}

	return count, iter.Error()
}

// Custom errors
var (
	ErrNotFound = fmt.Errorf("key not found")
)
