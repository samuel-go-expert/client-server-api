package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	data, statusCode, err := makeRequest()

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if statusCode != http.StatusOK {
		log.Printf("Status code: %d", statusCode)
		return
	}

	storeData(data)

	log.Printf("Data stored successfully")
}

func storeData(data float64) {

	file, err := os.OpenFile("data.txt", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
	defer file.Close()

	if err != nil {
		panic(err)
	}

	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("Dólar: {%f}\n", data))

	if err != nil {
		panic(err)
	}

}

func makeRequest() (float64, int, error) {
	url := "http://localhost:8080/cotacao"

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*5000)
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
		return 0, 0, ctx.Err()
	default:

		defer res.Body.Close()
		bodyResponse, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}

		temp := strings.ReplaceAll(string(bodyResponse), "\"", "")
		temp = strings.TrimSpace(temp)

		data, err := strconv.ParseFloat(temp, 64)

		if err != nil {
			panic(err)
		}

		return data, res.StatusCode, nil
	}
}
