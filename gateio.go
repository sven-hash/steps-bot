package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/antihax/optional"
	"github.com/gateio/gateapi-go/v6"
)

const (
	ONE_DAY_SEC = 86400
)

const (
	SL_BUY_RULE  string = ">="
	SL_SELL_RULE string = "<="
)

const MAX_ELEMENT_PAGE float64 = 100

func createOrder(pair string, side string, priceMin float64, priceMax float64, amount float64, steps float64, timeInForce string) []gateapi.Order {

	var orders []gateapi.Order

	numInter := math.Abs(priceMax-priceMin) / steps
	amountPerOrder := amount / math.Round(numInter)

	numOrders := 0
	currentPrice := priceMin
	totalAmountOrder := 0.0

	if amountPerOrder < 1.0 {
		fmt.Fprintf(os.Stderr, "\nAmount per order must be higher than 1 USDT\nIncrease the amount. Actual amount per order is %.3f\n", amountPerOrder)
		os.Exit(1)
	}

	fmt.Printf("%s %.5f USDT between %.5f and %.5f, amount per order: %.4f USDT\n", side, amount, priceMin, priceMax, amountPerOrder)
	for i := 0; i < int(numInter); i++ {
		alphAmount := amountPerOrder / currentPrice

		if amount-amountPerOrder <= 0 {
			break
		}

		order := gateapi.Order{
			Text:         generateId(10),
			CurrencyPair: pair,
			Side:         side,
			Price:        strconv.FormatFloat(round(currentPrice, 10000), 'f', -1, 64),
			Amount:       strconv.FormatFloat(alphAmount, 'f', -1, 64),
			TimeInForce:  timeInForce,
		}

		fmt.Printf("price: %.5f ALPH, amount: %.2f ALPH, total: %.4f USDT, amount left: %.2f USDT\n", round(currentPrice, 10000), alphAmount, alphAmount*currentPrice, amount)

		numOrders++
		currentPrice += steps
		amount -= alphAmount * currentPrice
		totalAmountOrder += alphAmount * currentPrice
		orders = append(orders, order)
	}

	fmt.Printf("Total amount in order: %.5f USDT\n", totalAmountOrder)
	return orders
}

func createOrderAlph(pair string, side string, priceMin float64, priceMax float64, amount float64, steps float64, timeInForce string) []gateapi.Order {

	var orders []gateapi.Order

	numInter := math.Abs(priceMax-priceMin) / steps
	amountPerOrder := amount / math.Round(numInter)

	numOrders := 0
	currentPrice := priceMin
	totalAmountOrder := 0.0

	if amountPerOrder*currentPrice < 1.0 {
		fmt.Fprintf(os.Stderr, "\nAmount per order must be higher than 1 USDT\nIncrease the amount. Actual amount per order is %.3f\n", amountPerOrder)
		os.Exit(1)
	}

	fmt.Printf("%s %.5f ALPH between %.5f and %.5f, amount per order: %.4f ALPH\n", side, amount, priceMin, priceMax, amountPerOrder)
	for i := 0; i < int(numInter); i++ {
		alphAmount := amountPerOrder

		if amount-amountPerOrder <= 0 {
			break
		}

		order := gateapi.Order{
			Text:         generateId(10),
			CurrencyPair: pair,
			Side:         side,
			Price:        strconv.FormatFloat(round(currentPrice, 10000), 'f', -1, 64),
			Amount:       strconv.FormatFloat(alphAmount, 'f', -1, 64),
			TimeInForce:  timeInForce,
		}

		fmt.Printf("price: %.5f USDT, amount: %.2f ALPH, total: %.4f USDT, amount left: %.2f ALPH\n", round(currentPrice, 10000), alphAmount, alphAmount, amount)

		numOrders++
		currentPrice += steps
		amount -= alphAmount
		totalAmountOrder += alphAmount
		orders = append(orders, order)
	}

	fmt.Printf("Total amount in order: %.5f USDT\n", totalAmountOrder)
	return orders
}

func createTriggeredOrder(pair string, side string, priceMin float64, priceMax float64, amount float64, steps float64) []gateapi.SpotPriceTriggeredOrder {
	var orders []gateapi.SpotPriceTriggeredOrder

	numInter := math.Abs(priceMax-priceMin) / steps
	amountPerOrder := amount / math.Round(numInter)

	numOrders := 0
	currentPrice := priceMin
	totalAmountOrder := 0.0
	var expirationSec int32 = ONE_DAY_SEC //one day

	rule := SL_BUY_RULE
	if side == sell {
		rule = SL_SELL_RULE
	}

	if amountPerOrder < 1.0 {
		fmt.Fprintf(os.Stderr, "\nAmount per order must be higher than 1 USDT\nIncrease the amount. Actual amount per order is %.3f\n", amountPerOrder)
		os.Exit(1)
	}

	fmt.Printf("%s %.5f USDT between %.5f and %.5f, amount per order: %.4f USDT, duration: %d day\n", side, amount, priceMin, priceMax, amountPerOrder, expirationSec/ONE_DAY_SEC)
	for i := 0; i < int(numInter); i++ {
		alphAmount := round(amountPerOrder/currentPrice, 1000)

		if amount-amountPerOrder <= 0 {
			break
		}

		order := gateapi.SpotPriceTriggeredOrder{
			Market: pair,
			Put: gateapi.SpotPricePutOrder{
				Type:        "limit",
				Side:        side,
				Price:       strconv.FormatFloat(round(currentPrice, 10000), 'f', -1, 64),
				Amount:      strconv.FormatFloat(alphAmount, 'f', -1, 64),
				Account:     "normal",
				TimeInForce: "ioc",
			},
			Trigger: gateapi.SpotPriceTrigger{
				Price:      strconv.FormatFloat(round(currentPrice-0.001, 10000), 'f', -1, 64),
				Rule:       rule,
				Expiration: expirationSec,
			},
		}

		fmt.Printf("price: %.5f ALPH, Stop price: %.4f USDT, amount: %.4f ALPH, total: %.4f USDT, amount left: %.4f USDT\n", round(currentPrice, 10000), round(currentPrice-0.001, 10000), alphAmount, alphAmount*currentPrice, amount)

		numOrders++
		currentPrice += steps
		amount -= alphAmount * currentPrice
		totalAmountOrder += alphAmount * currentPrice
		orders = append(orders, order)
	}

	fmt.Printf("Total amount in order: %.5f USDT\n", totalAmountOrder)
	return orders

}

func sendOrder(client *gateapi.APIClient, ctx *context.Context, orders gateapi.Order) {

	result, _, err := client.SpotApi.CreateOrder(*ctx, orders)
	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
		}
	} else {
		fmt.Println(result)
	}

}

func getOrders(client *gateapi.APIClient, ctx *context.Context, pair string, status string, options *gateapi.ListOrdersOpts) []gateapi.Order {
	result, _, err := client.SpotApi.ListOrders(*ctx, pair, status, options)
	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
		}
	} else {
		//fmt.Printf("%+v\n", result)
	}

	return result
}

func checkBalance(client *gateapi.APIClient, ctx *context.Context) []gateapi.SpotAccount {

	result, _, err := client.SpotApi.ListSpotAccounts(*ctx, nil)
	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
			panic(e)
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
			panic(e)

		}
	}

	return result

}

func sendBatchOrder(client *gateapi.APIClient, ctx *context.Context, orders []gateapi.Order) {

	result, _, err := client.SpotApi.CreateBatchOrders(*ctx, orders)
	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
		}
	} else {

		for resultIndex := 0; resultIndex < len(result); resultIndex++ {
			if result[resultIndex].Status == "filled" {
				orderFilled := result[resultIndex]
				fmt.Printf("Order get filled: price: %s USDT, amount: %s ALPH, total: %s USDT\n", orderFilled.FillPrice, orderFilled.Amount, orderFilled.FillPrice)
			}
		}
	}

}

func sendTriggeredOrder(client *gateapi.APIClient, ctx *context.Context, spotPriceTriggeredOrder *gateapi.SpotPriceTriggeredOrder) {

	result, _, err := client.SpotApi.CreateSpotPriceTriggeredOrder(*ctx, *spotPriceTriggeredOrder)
	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
		}
	} else {
		fmt.Println(result)
	}
}

func getOpenOrders(client *gateapi.APIClient, ctx *context.Context, currency_pair string) []gateapi.Order {
	result, _, err := client.SpotApi.ListAllOpenOrders(*ctx, &gateapi.ListAllOpenOrdersOpts{Limit: optional.NewInt32(100), Page: optional.NewInt32(1)})

	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
		}
	}

	if (len(result)) > 0 {
		for pairIndex := 0; pairIndex < len(result); pairIndex++ {
			if result[pairIndex].CurrencyPair == currency_pair {
				return result[pairIndex].Orders
			}
		}
	}
	return []gateapi.Order{}
}

func getTickerPrice(client *gateapi.APIClient, ctx *context.Context, pair string) float64 {
	result, _, err := client.SpotApi.ListTickers(*ctx, &gateapi.ListTickersOpts{CurrencyPair: optional.NewString(pair)})
	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
		}
	}

	if len(result) > 0 {
		price, err := strconv.ParseFloat(result[0].Last, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot get price: %s", err)
		}

		return price
	}
	return 0.0
}

func getAccountDetails(client *gateapi.APIClient, ctx *context.Context) {
	result, _, err := client.AccountApi.GetAccountDetail(*ctx)
	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
			panic(e)
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
			panic(e)
		}
	} else {
		fmt.Printf("%+v\n", result)
	}
}
