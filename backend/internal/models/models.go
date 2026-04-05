package models

import (
	"time"

	"github.com/google/uuid"
	coremodels "github.com/ya-breeze/kin-core/models"
	"gorm.io/gorm"
)

type Family struct {
	coremodels.Family
	Currency string         `gorm:"default:'₽'" json:"currency"`
	Users    []User         `gorm:"foreignKey:FamilyID" json:"users"`
	Lists    []ShoppingList `gorm:"foreignKey:FamilyID" json:"lists"`
	Shops    []Shop         `gorm:"foreignKey:FamilyID" json:"shops"`
}

type User struct {
	coremodels.User
}

type ShoppingList struct {
	coremodels.TenantModel
	Title           string     `gorm:"not null" json:"title"`
	Status          string     `json:"status"` // "planning", "in-progress", "completed"
	EstimatedAmount float64    `json:"estimated_amount"`
	ActualAmount    float64    `json:"actual_amount"`
	CompletedAt     *time.Time `json:"completed_at"`
	Items           []Item     `gorm:"foreignKey:ListID" json:"items"`
	Receipts        []Receipt  `gorm:"foreignKey:ListID" json:"receipts"`
}

type Item struct {
	coremodels.TenantModel
	Name           string     `gorm:"not null" json:"name"`
	Description    string     `json:"description"`
	Quantity       float64    `gorm:"default:1" json:"quantity"`
	Unit           string     `gorm:"default:'pcs'" json:"unit"` // "pcs", "kg", "100g", etc.
	IsBought       bool       `gorm:"default:false" json:"is_bought"`
	Price          float64    `json:"price"`
	LocalPhotoPath string     `json:"local_photo_path"`
	IsUrgent       bool       `gorm:"default:false" json:"is_urgent"`
	ListID         uuid.UUID  `gorm:"type:uuid;not null" json:"list_id"`
	CategoryID     uuid.UUID  `gorm:"type:uuid" json:"category_id"`
	FlyerItemID    *uint      `json:"flyer_item_id"`
	ReceiptItemID  *uint      `json:"receipt_item_id"`
}

type Category struct {
	coremodels.TenantModel
	Name      string `gorm:"not null" json:"name"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
}

type Shop struct {
	coremodels.TenantModel
	Name string `gorm:"not null" json:"name"`
}

type ShopCategoryOrder struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ShopID     uuid.UUID `gorm:"type:uuid" json:"shop_id"`
	CategoryID uuid.UUID `gorm:"type:uuid" json:"category_id"`
	SortOrder  int       `json:"sort_order"`
}

type ItemFrequency struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FamilyID  uuid.UUID `gorm:"type:uuid" json:"family_id"`
	ItemName  string    `gorm:"not null" json:"item_name"`
	Frequency int       `gorm:"default:1" json:"frequency"`
	LastPrice float64   `json:"last_price"`
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
	ShopName  string    `gorm:"->;column:shop_name" json:"shop_name"`
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
	SearchText     string         `gorm:"index" json:"-"`
	ShopName       string         `gorm:"->;column:shop_name" json:"shop_name"`
}

type JobStatus struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `gorm:"uniqueIndex" json:"name"`
	LastRun   time.Time `json:"last_run"`
}

type Receipt struct {
	coremodels.TenantModel
	ListID    *uuid.UUID    `gorm:"type:uuid" json:"list_id"`
	ShopID    *uuid.UUID    `gorm:"type:uuid" json:"shop_id"` // Optional, if matched to a shop
	Shop      *Shop         `gorm:"foreignKey:ShopID" json:"shop"`
	Date      time.Time     `json:"date"`
	Total     float64       `json:"total"`
	ImagePath string        `json:"image_path"`                  // Path relative to kincart-data
	Status    string        `gorm:"default:'new'" json:"status"` // "new", "parsed", "error"
	Items     []ReceiptItem `gorm:"foreignKey:ReceiptID" json:"items"`
}

type ReceiptItem struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ReceiptID      uuid.UUID `gorm:"type:uuid;not null" json:"receipt_id"`
	Name           string    `json:"name"`
	Quantity       float64   `json:"quantity"`
	Unit           string    `json:"unit"` // e.g., "pcs", "kg"
	Price          float64   `json:"price"`
	TotalPrice     float64   `json:"total_price"`
	MatchedItemID  *uuid.UUID `gorm:"type:uuid" json:"matched_item_id"` // planned Item it was matched to
	MatchStatus    string    `gorm:"default:'unmatched'" json:"match_status"` // "auto","confirmed","manual","unmatched","dismissed"
	Confidence     int       `json:"confidence"`       // 0-100
	SuggestedItems string    `json:"suggested_items"`  // JSON: [{"item_id":"uuid","item_name":"jogurt","confidence":85}]
}

// ItemAlias records the mapping between a generic planned item name and the
// specific receipt name it was bought as, building per-family purchase history.
type ItemAlias struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	FamilyID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"family_id"`
	PlannedName   string     `gorm:"not null" json:"planned_name"`  // e.g. "jogurt"
	ReceiptName   string     `gorm:"not null" json:"receipt_name"`  // e.g. "selský jogurt 2%"
	ShopID        *uuid.UUID `gorm:"type:uuid" json:"shop_id"`
	Shop          *Shop      `gorm:"foreignKey:ShopID" json:"shop"`
	LastPrice     float64    `json:"last_price"`
	PurchaseCount int        `gorm:"default:1" json:"purchase_count"`
	LastUsedAt    time.Time  `json:"last_used_at"`
	CreatedAt     time.Time  `json:"created_at"`
}
