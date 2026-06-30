package model

type User struct {
	ID           string `gorm:"primaryKey"`
	Username     string `gorm:"unique;not null"`
	PasswordHash string `gorm:"not null"`
	Email        string
	CreatedAt    int64
}
