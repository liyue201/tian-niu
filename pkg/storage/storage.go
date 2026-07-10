package storage

import "context"

type Storage interface {
	Load(ctx context.Context, key string) (string, error)
	Store(ctx context.Context, key string, value string) error
}

type AdvancedStorage interface {
	Storage
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
}
