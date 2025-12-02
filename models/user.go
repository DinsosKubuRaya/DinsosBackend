package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID       string `gorm:"type:char(36);primaryKey" json:"id"`
	Name     string `gorm:"type:varchar(100)" json:"name"`
	Username string `gorm:"type:varchar(100);unique" json:"username"`
	Password string `gorm:"type:varchar(255)" json:"password"`
	Role     string `gorm:"type:enum('admin','staff','superadmin')" json:"role"`

	// TAMBAH FIELD INI
	PhotoURL *string `gorm:"type:text;default:null" json:"photo_url"`
	PhotoID  *string `gorm:"type:varchar(255);default:null" json:"photo_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Generate UUID sebelum disimpan
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	// Pastikan ini 'u.ID' (huruf besar)
	u.ID = uuid.NewString()
	return
}
