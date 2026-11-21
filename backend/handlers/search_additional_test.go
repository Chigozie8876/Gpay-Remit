package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Gpay-Remit/gpay-remit/config"
	"github.com/Gpay-Remit/gpay-remit/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSearchByAmountCurrencyStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB()
	cfg := &config.Config{}

	// Seed payments
	db.Create(&models.Payment{Amount: 1550, Currency: "USD", Status: "completed", Notes: "Lunch"})
	db.Create(&models.Payment{Amount: 5000, Currency: "EUR", Status: "pending", Notes: "Invoice"})
	db.Create(&models.Payment{Amount: 5000, Currency: "USD", Status: "failed", Notes: "Refund"})

	handler := NewSearchHandler(db, cfg)
	router := gin.New()
	router.GET("/search/transactions", handler.SearchTransactions)

	// Search by amount
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/search/transactions?q=5000", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	// Expect two results with amount 5000
	assert.Len(t, data, 2)

	// Search by currency
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/search/transactions?q=EUR", nil)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	var resp2 map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	data2 := resp2["data"].([]interface{})
	assert.Len(t, data2, 1)

	// Search by status
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/search/transactions?q=failed", nil)
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
	var resp3 map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &resp3)
	data3 := resp3["data"].([]interface{})
	assert.Len(t, data3, 1)
}

func TestSearchRejectsSQLInjectionPayloadAndLongQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB()
	handler := NewSearchHandler(db, &config.Config{})
	router := gin.New()
	router.Use(errorHandlerForTests())
	router.GET("/search/transactions", handler.SearchTransactions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/search/transactions?q=%27%3B+DROP+TABLE+payments+--", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/search/transactions?q="+strings.Repeat("a", maxSearchQueryLength+1), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
