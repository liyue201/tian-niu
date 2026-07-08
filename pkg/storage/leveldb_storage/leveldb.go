package leveldb_storage

import (
	"context"
	"errors"

	"github.com/syndtr/goleveldb/leveldb"
)

type LevelDBStorage struct {
	db *leveldb.DB
}

func NewLevelDBStorage(path string) (*LevelDBStorage, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &LevelDBStorage{
		db: db,
	}, nil
}

func (s *LevelDBStorage) Load(ctx context.Context, key string) (string, error) {
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func (s *LevelDBStorage) Store(ctx context.Context, key string, value string) error {
	return s.db.Put([]byte(key), []byte(value), nil)
}

func (s *LevelDBStorage) Close() error {
	return s.db.Close()
}
