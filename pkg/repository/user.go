package repository

import (
	"github.com/liyue201/tian-niu/pkg/model"
)

func (r *Repository) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
