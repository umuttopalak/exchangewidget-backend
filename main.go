package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	socket "github.com/umuttopalak/tradingview-scraper"
)

type SymbolData struct {
	Current          *socket.QuoteData `json:"current"`
	Previous         *socket.QuoteData `json:"previous"`
	PercentageChange *float64          `json:"percentage_change,omitempty"`
}

var (
	latestData        = make(map[string]*SymbolData)
	tradingviewsocket socket.SocketInterface
	dataMutex         sync.Mutex
)

func main() {
	go connectAndRetry()
	go periodicTask() // Yeni eklenen goroutine

	r := gin.Default()

	r.GET("/api/symbol-data", getAllSymbolData)
	r.GET("/api/symbol-data/:symbol", getSymbolData)
	r.POST("/api/symbol", addSymbol)
	r.DELETE("/api/symbol/:symbol", removeSymbol)

	r.Run(":8080")
}

func connectAndRetry() {
	for {
		err := connectToWebSocket()
		if err != nil {
			fmt.Printf("Error: %s. Retrying in 5 seconds...\n", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func periodicTask() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Println("10 dakikalık periyodik görev tetiklendi.")
			performPeriodicOperation()
		}
	}
}

func performPeriodicOperation() {
	// Örneğin, tüm sembolleri yeniden bağlamak
	dataMutex.Lock()
	defer dataMutex.Unlock()
	for symbol := range latestData {
		tradingviewsocket.AddSymbol(symbol)
	}
	fmt.Println("Periyodik operasyon gerçekleştirildi.")
}

func adjustPriceWithScale(price float64, priceScale int) float64 {
	scaleFactor := float64(priceScale)
	return price / scaleFactor
}

func connectToWebSocket() error {
	var err error
	tradingviewsocket, err = socket.Connect(
		func(symbol string, data *socket.QuoteData) {
			dataMutex.Lock()
			defer dataMutex.Unlock()

			if existingData, exists := latestData[symbol]; exists {
				existingData.Previous = existingData.Current
				existingData.Current = data

				if data.Price != nil && data.PriceScale != nil {
					adjustedPrice := adjustPriceWithScale(*data.Price, int(*data.PriceScale))
					*data.Price = adjustedPrice
				}

				if existingData.Previous != nil && existingData.Previous.Price != nil && existingData.Current.Price != nil {
					percentageChange := calculatePercentageChange(*existingData.Previous.Price, *existingData.Current.Price)
					existingData.PercentageChange = &percentageChange
				} else {
					existingData.PercentageChange = nil
				}
			} else {
				if data.Price != nil && data.PriceScale != nil {
					adjustedPrice := adjustPriceWithScale(*data.Price, int(*data.PriceScale))
					*data.Price = adjustedPrice
				}

				latestData[symbol] = &SymbolData{
					Current:  data,
					Previous: nil,
				}
			}
		},
		func(err error, context string) {
			fmt.Printf("Error: %s, Context: %s\n", err.Error(), context)
		},
	)

	if err != nil {
		return err
	}

	for {
		time.Sleep(30 * time.Second)
	}

	return nil
}

func calculatePercentageChange(previousPrice, currentPrice float64) float64 {
	if previousPrice == 0 {
		return 0
	}
	return ((currentPrice - previousPrice) / previousPrice) * 100
}

func getAllSymbolData(c *gin.Context) {
	dataMutex.Lock()
	defer dataMutex.Unlock()

	response, err := json.Marshal(latestData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to fetch data"})
		return
	}

	c.Data(http.StatusOK, "application/json", response)
}

func getSymbolData(c *gin.Context) {
	symbol := c.Param("symbol")

	dataMutex.Lock()
	defer dataMutex.Unlock()

	data, exists := latestData[symbol]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Symbol not found"})
		return
	}

	response, err := json.Marshal(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to fetch data"})
		return
	}

	c.Data(http.StatusOK, "application/json", response)
}

func addSymbol(c *gin.Context) {
	var request struct {
		Symbol string `json:"symbol"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	tradingviewsocket.AddSymbol(request.Symbol)

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Symbol %s added", request.Symbol)})
}

func removeSymbol(c *gin.Context) {
	symbol := c.Param("symbol")

	dataMutex.Lock()
	defer dataMutex.Unlock()

	if _, exists := latestData[symbol]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Symbol not found"})
		return
	}

	tradingviewsocket.RemoveSymbol(symbol)
	delete(latestData, symbol)

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Symbol %s removed", symbol)})
}
