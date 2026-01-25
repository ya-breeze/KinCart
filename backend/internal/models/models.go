package models

import (
	"time"

	"gorm.io/gorm"
)

type Family struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"not null" json:"name"`
	Currency  string         `gorm:"default:'₽'" json:"currency"`
	Users     []User         `gorm:"foreignKey:FamilyID" json:"users"`
	Lists     []ShoppingList `gorm:"foreignKey:FamilyID" json:"lists"`
	Shops     []Shop         `gorm:"foreignKey:FamilyID" json:"shops"`
}

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Username     string         `gorm:"unique;not null" json:"username"`
	PasswordHash string         `json:"-"`
	FamilyID     uint           `json:"family_id"`
	Family       Family         `gorm:"foreignKey:FamilyID" json:"family,omitempty"`
}

type ShoppingList struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Title           string         `gorm:"not null" json:"title"`
	Status          string         `json:"status"` // "planning", "in-progress", "completed"
	EstimatedAmount float64        `json:"estimated_amount"`
	ActualAmount    float64        `json:"actual_amount"`
	CompletedAt     *time.Time     `json:"completed_at"`
	FamilyID        uint           `json:"family_id"`
	Items           []Item         `gorm:"foreignKey:ListID" json:"items"`
}

type Item struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Name           string         `gorm:"not null" json:"name"`
	Description    string         `json:"description"`
	Quantity       float64        `gorm:"default:1" json:"quantity"`
	Unit           string         `gorm:"default:'pcs'" json:"unit"` // "pcs", "kg", "100g", etc.
	IsBought       bool           `gorm:"default:false" json:"is_bought"`
	Price          float64        `json:"price"`
	LocalPhotoPath string         `json:"local_photo_path"`
	IsUrgent       bool           `gorm:"default:false" json:"is_urgent"`
	ListID         uint           `json:"list_id"`
	CategoryID     uint           `json:"category_id"`
}

type Category struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"not null" json:"name"`
	Icon      string         `json:"icon"`
	SortOrder int            `json:"sort_order"`
	FamilyID  uint           `json:"family_id"`
}

type Shop struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"not null" json:"name"`
	FamilyID  uint           `json:"family_id"`
}

type ShopCategoryOrder struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	ShopID     uint `json:"shop_id"`
	CategoryID uint `json:"category_id"`
	SortOrder  int  `json:"sort_order"`
}

type ItemFrequency struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	FamilyID  uint   `json:"family_id"`
	ItemName  string `gorm:"not null" json:"item_name"`
	Frequency int    `gorm:"default:1" json:"frequency"`
}
