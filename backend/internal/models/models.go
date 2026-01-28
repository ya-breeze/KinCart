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
	FlyerItemID    *uint          `json:"flyer_item_id"`
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

type Flyer struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	ShopName  string         `json:"shop_name"`
	URL       string         `gorm:"index" json:"url"`
	StartDate time.Time      `json:"start_date"`
	EndDate   time.Time      `json:"end_date"`
	ParsedAt  time.Time      `json:"parsed_at"`
	Pages     []FlyerPage    `gorm:"foreignKey:FlyerID" json:"pages"`
}

type FlyerPage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	FlyerID   uint      `json:"flyer_id"`
	SourceURL string    `json:"source_url"`
	LocalPath string    `json:"local_path"`
	IsParsed  bool      `gorm:"default:false" json:"is_parsed"`
	Retries   int       `gorm:"default:0" json:"retries"`
	LastError string    `json:"last_error"`
}

type FlyerItem struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	FlyerID        uint           `json:"flyer_id"`
	FlyerPageID    uint           `json:"flyer_page_id"`
	Name           string         `json:"name"`
	Price          float64        `json:"price"`
	OriginalPrice  *float64       `json:"original_price"`
	Quantity       string         `json:"quantity"` // e.g., "1kg", "100g", "pcs"
	StartDate      time.Time      `json:"start_date"`
	EndDate        time.Time      `json:"end_date"`
	PhotoURL       string         `json:"photo_url"`
	LocalPhotoPath string         `json:"local_photo_path"`
	Categories     string         `json:"categories"` // comma-separated English categories
	Keywords       string         `json:"keywords"`   // comma-separated English keywords
	ShopName       string         `gorm:"->;column:shop_name" json:"shop_name"`
}

type JobStatus struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `gorm:"uniqueIndex" json:"name"`
	LastRun   time.Time `json:"last_run"`
}
