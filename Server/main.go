package main

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ExchangeRate struct {
	Bid        string `json:"bid"`
	CreateDate string `json:"create_date"`
	gorm.Model
}

type Response struct {
	ExchangeRate ExchangeRate `json:"USDBRL"`
}

func main() {

	mux := http.NewServeMux()
	db := dbStart(&ExchangeRate{})

	mux.Handle("/cotacao", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		exchangeRateHandler(writer, request, db)
	}))

	err := http.ListenAndServe(":8080", mux)

	if err != nil {
		return
	}
}

func exchangeRateHandler(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	data, err := makeRequest()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = dbInsert(db, data)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	println(data.Bid)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(data.Bid)

	if err != nil {
		panic(err)
	}
}

func makeRequest() (*ExchangeRate, error) {

	url := "https://economia.awesomeapi.com.br/json/last/USD-BRL"

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		panic(err)
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			panic(err)
		}
	}

	select {
	case <-ctx.Done():
		log.Printf("Http request canceled: %v", ctx.Err())
		return nil, ctx.Err()
	default:
		defer res.Body.Close()

		bodyResponse, err := io.ReadAll(res.Body)

		if err != nil {
			panic(err)
		}

		var data Response

		err = json.Unmarshal(bodyResponse, &data)

		data.ExchangeRate.Bid = strings.Replace(data.ExchangeRate.Bid, ",", ".", -1)

		return &data.ExchangeRate, nil
	}

}

func dbStart(dst ...interface{}) *gorm.DB {

	file, err := os.OpenFile("exchange.sqlite", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
	file.Close()

	db, err := gorm.Open(sqlite.Open("exchange.sqlite"), &gorm.Config{})

	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(dst...)

	if err != nil {
		panic(err)
	}

	return db
}

func dbInsert(db *gorm.DB, data *ExchangeRate) error {

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*300)
	defer cancel()

	db.WithContext(ctx).Create(&ExchangeRate{
		Bid:        data.Bid,
		CreateDate: data.CreateDate,
	})

	select {
	case <-ctx.Done():
		log.Printf("Database insert canceled: %s", ctx.Err())
		return ctx.Err()
	default:
		return nil
	}

}
