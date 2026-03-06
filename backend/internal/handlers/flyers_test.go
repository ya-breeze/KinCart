package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kincart/internal/database"
	"kincart/internal/models"
	"kincart/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupFlyerTestDB() {
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}
	database.DB.AutoMigrate(&models.Flyer{}, &models.FlyerItem{}, &models.FlyerPage{})
}

func TestGetFlyerItemsSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupFlyerTestDB()

	// Create test data
	now := time.Now()
	flyer := models.Flyer{
		ShopName:  "Test Shop",
		StartDate: now.AddDate(0, 0, -1),
		EndDate:   now.AddDate(0, 0, 1),
	}
	database.DB.Create(&flyer)

	items := []models.FlyerItem{
		{
			FlyerID:    flyer.ID,
			Name:       "Myčka nádobí",
			Categories: "Kuchyně",
			Keywords:   "spotřebič",
			SearchText: utils.NormalizeSearchText("Myčka nádobí Kuchyně spotřebič"),
			StartDate:  flyer.StartDate,
			EndDate:    flyer.EndDate,
		},
		{
			FlyerID:    flyer.ID,
			Name:       "Sušenky",
			Categories: "Sladkosti",
			Keywords:   "ke kávě",
			SearchText: utils.NormalizeSearchText("Sušenky Sladkosti ke kávě"),
			StartDate:  flyer.StartDate,
			EndDate:    flyer.EndDate,
		},
	}
	for _, item := range items {
		database.DB.Create(&item)
	}

	r := gin.New()
	r.GET("/flyers/items", GetFlyerItems)

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectedFirst string
	}{
		{"Exact match", "Myčka", 1, "Myčka nádobí"},
		{"Diacritic-insensitive match", "mycka", 1, "Myčka nádobí"},
		{"Partial match", "nádobí", 1, "Myčka nádobí"},
		{"Partial diacritic-insensitive match", "nadobi", 1, "Myčka nádobí"},
		{"Category match", "Kuchyne", 1, "Myčka nádobí"},
		{"Keyword match", "kave", 1, "Sušenky"},
		{"No match", "nonexistent", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/flyers/items?q="+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			var resp struct {
				Items []models.FlyerItem `json:"items"`
			}
			json.Unmarshal(w.Body.Bytes(), &resp)
			assert.Equal(t, tt.expectedCount, len(resp.Items))
			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedFirst, resp.Items[0].Name)
			}
		})
	}
}
