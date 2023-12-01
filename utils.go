package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gateio/gateapi-go/v6"
)

const (
	buy  string = "buy"
	sell string = "sell"
)

// from the ticker choose if it's crypto or fiat
func selectFiatOrCrypto(ticker string, pair string, side string, priceMin float64, priceMax float64, amount float64, steps float64, timeInForce string) []gateapi.Order {

	if strings.ToUpper(ticker) == "USDT" {
		return createOrder("ALPH_USDT", side, priceMin, priceMax, amountUSDT, steps, timeInForce)
	}

	return createOrderAlph("ALPH_USDT", side, priceMin, priceMax, amountAlph, steps, timeInForce)
}

func selectFiatOrCryptoTriggered(ticker string, pair string, side string, priceMin float64, priceMax float64, amount float64, steps float64) []gateapi.SpotPriceTriggeredOrder {

	if strings.ToUpper(ticker) == "USDT" {
		return createTriggeredOrder("ALPH_USDT", side, priceMin, priceMax, amountUSDT, steps)
	}

	return []gateapi.SpotPriceTriggeredOrder{}
}

func checkOrdersOpen(client *gateapi.APIClient, ctx *context.Context, pair string) bool {
	return len(getOpenOrders(client, ctx, pair)) > 0
}

func generateId(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" + "_-."
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return "t-" + string(b)
}

func round(x, unit float64) float64 {
	return math.Round(x*unit) / unit
}

func formatOpenOrders(order *gateapi.Order) {

	amount, err := strconv.ParseFloat(order.Amount, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Formatting not possible for formatOpenOrders: %s\n", err)
		panic(err)
	}

	filled, err := strconv.ParseFloat(order.FilledTotal, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Formatting not possible for formatOpenOrders: %s\n", err)
		panic(err)
	}

	price, err := strconv.ParseFloat(order.Price, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Formatting not possible for formatOpenOrders: %s\n", err)
		panic(err)
	}
	amountFiat := price * amount

	percentLeft := 0.0
	if filled > 0 {
		percentLeft = amount / filled
	}

	fmt.Printf("Price: %s USDT, Volume: %.3f ALPH | %.3f USDT, Filled Total: %.3f ALPH (%.2f %%)\n", order.Price, amount, amountFiat, filled, percentLeft)
}

func median(data []float64) float64 {
	dataCopy := make([]float64, len(data))
	copy(dataCopy, data)

	sort.Float64s(dataCopy)

	var median float64
	l := len(dataCopy)
	if l == 0 {
		return 0
	} else if l%2 == 0 {
		median = (dataCopy[l/2-1] + dataCopy[l/2]) / 2
	} else {
		median = dataCopy[l/2]
	}

	return median
}
