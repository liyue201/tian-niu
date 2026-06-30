package repository

import (
	"github.com/libtnb/sqlite"
	"github.com/liyue201/tian-niu/pkg/model"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(dsn string) (*Repository, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&model.User{}, &model.Conversation{}, &model.ChatMessage{})
	if err != nil {
		return nil, err
	}
	return &Repository{
		db: db,
	}, nil
}

func (r *Repository) Create(v interface{}) error {
	return r.db.Create(v).Error
}

func (r *Repository) Delete(v interface{}) error {
	return r.db.Delete(v).Error
}
