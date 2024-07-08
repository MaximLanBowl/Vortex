package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DepthOrder struct {
	Price    float64 `json:"price"`
	BaseQty  float64 `json:"base_qty"`
}

type HistoryOrder struct {
	ClientName              string    `json:"client_name"`
	ExchangeName            string    `json:"exchange_name"`
	Label                   string    `json:"label"`
	Pair                     string    `json:"pair"`
	Side                     string    `json:"side"`
	Type                     string    `json:"type"`
	BaseQty                  float64   `json:"base_qty"`
	Price                    float64   `json:"price"`
	AlgorithmNamePlaced      string    `json:"algorithm_name_placed"`
	LowestSellPrc            float64   `json:"lowest_sell_prc"`
	HighestBuyPrc            float64   `json:"highest_buy_prc"`
	CommissionQuoteQty       float64   `json:"commission_quote_qty"`
	TimePlaced               time.Time `json:"time_placed"`
}

type Client struct {
	ClientName string `json:"client_name"`
	ExchangeName string `json:"exchange_name"`
	Label string `json:"label"`
	Pair string `json:"pair"`
}

type App struct {
	DB *sqlx.DB
}

func main() {
	db, err := sqlx.Connect("postgres", "user=postgres password=password dbname=mydb sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	app := &App{DB: db}

	r := gin.Default()
	r.GET("/order-book/:exchange/:pair", app.GetOrderBook)
	r.POST("/order-book/:exchange/:pair", app.SaveOrderBook)
	r.GET("/order-history/:client_name/:exchange_name/:label/:pair", app.GetOrderHistory)
	r.POST("/order-history/:client_name/:exchange_name/:label/:pair", app.SaveOrder)

	r.Run()
}

func (a *App) GetOrderBook(c *gin.Context) {
	exchangeName := c.Param("exchange")
	pair := c.Param("pair")

	var orderBook []DepthOrder
	err := a.DB.Select(&orderBook, "SELECT * FROM order_book WHERE exchange = $1 AND pair = $2", exchangeName, pair)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orderBook)
}

func (a *App) SaveOrderBook(c *gin.Context) {
	exchangeName := c.Param("exchange")
	pair := c.Param("pair")

	var orderBook []DepthOrder
	if err := c.BindJSON(&orderBook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := a.DB.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec("DELETE FROM order_book WHERE exchange = $1 AND pair = $2", exchangeName, pair)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, order := range orderBook {
		_, err = tx.Exec("INSERT INTO order_book (id, exchange, pair, price, base_qty) VALUES ($1, $2, $3, $4, $5)", uuid.New().String(), exchangeName, pair, order.Price, order.BaseQty)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order book saved successfully"})
}

func (a *App) GetOrderHistory(c *gin.Context) {
	clientName := c.Param("client_name")
	exchangeName := c.Param("exchange_name")
	label := c.Param("label")
	pair := c.Param("pair")

	var orderHistory []HistoryOrder
	err := a.DB.Select(&orderHistory, "SELECT * FROM order_history WHERE client_name = $1 AND exchange_name = $2 AND label = $3 AND pair = $4", clientName, exchangeName, label, pair)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orderHistory)
}

// func (a *App) SaveOrder(c *gin.Context) {
// 	clientName := c.Param("client_name")
// 	exchangeName := c.Param("exchange_name")
// 	label := c.Param("label")
// 	pair := c.Param("pair")

// 	var order HistoryOrder
// 	if err := c.BindJSON(&order); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	_, err := a.DB.Exec("INSERT INTO order_history (client_name, exchange_name, label, pair, side, type, base_qty, price, algorithm_name_placed, lowest_sell_prc, highest_buy_prc, commission_quote_qty, time_placed) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)", order.ClientName, order.ExchangeName, order.Label, order.Pair, order.Side, order.Type, order.BaseQty, order.Price, order.AlgorithmNamePlaced, order.LowestSellPrc, order.HighestBuyPrc, order.CommissionQuoteQty, order.TimePlaced)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Order saved successfully"})
// }