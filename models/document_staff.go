package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DocumentStaff struct {
	ID     string `gorm:"type:char(36);primaryKey" json:"id"`
	UserID string `gorm:"type:char(36);not null" json:"user_id"`
	User   User   `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user"`

	Sender       string `gorm:"type:varchar(255)" json:"sender"`
	Subject      string `gorm:"type:varchar(255)" json:"subject"`
	LetterType   string `gorm:"type:varchar(20)" json:"letter_type"` // masuk/keluar
	FileName     string `gorm:"type:varchar(500)" json:"file_name"`  // URL Cloudinary
	PublicID     string `gorm:"type:varchar(255)" json:"public_id"`
	ResourceType string `gorm:"type:varchar(20)" json:"resource_type"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (d *DocumentStaff) BeforeCreate(tx *gorm.DB) (err error) {
	d.ID = uuid.NewString()
	return
}
