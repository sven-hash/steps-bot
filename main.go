package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/antihax/optional"
	"github.com/gateio/gateapi-go/v6"
	"github.com/joho/godotenv"
)

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

const GATE_MAX_SIZE_BATCH int = 10
const DEFAULT_STEPS float64 = 0.005

const (
	GOOD_TILL_CANCEL    = "gtc"
	IMMEDIATE_OR_CANCEL = "ioc"
)

var priceMin float64
var priceMax float64
var amountUSDT float64
var amountAlph float64
var steps float64
var side string
var listOpenOrders bool
var timeInForce string
var useSl bool
var listPastOrders bool
var limit int64
var lastDays int

var gateioKey string
var gateioSecret string

func getParams() {
	flag.Float64Var(&priceMin, "min", 0.0, "Define minimum price")

	flag.Float64Var(&priceMax, "max", 0.0, "Define maximum price")

	flag.StringVar(&side, "side", "", "buy or sell")

	flag.Float64Var(&amountUSDT, "amountUsdt", 0.0, "Set the total amount in USDT")

	flag.Float64Var(&amountAlph, "amountAlph", 0.0, "Set the total amount in USDT")

	flag.Float64Var(&steps, "steps", DEFAULT_STEPS, "Set the steps between the prices")

	flag.BoolVar(&listOpenOrders, "listopen", false, "List open orders")

	flag.BoolVar(&listPastOrders, "list", false, "List open orders")
	flag.Int64Var(&limit, "limit", 10, "set number of order to check")
	flag.IntVar(&lastDays, "lastdays", 0, "set the n last days of trades")

	flag.StringVar(&timeInForce, "timeinforce", GOOD_TILL_CANCEL, "Time in for, good till (gtc) can or immediate or cancel (ioc)")

	flag.BoolVar(&useSl, "sl", false, "Use Stop-Limit instead of Limit")

	flag.Parse()
	checkArgs()
}

func checkArgs() {
	error := false

	if listOpenOrders || listPastOrders {
		return
	}

	if priceMin <= 0.0 {
		fmt.Fprintf(os.Stderr, "min argument is mandatory\n")
		error = true
	}

	if priceMax <= 0.0 {
		fmt.Fprintf(os.Stderr, "max argument is mandatory\n")
		error = true
	}

	if priceMin >= priceMax {
		fmt.Fprintf(os.Stderr, "min cannot be higher than max\n")
		error = true
	}

	if amountUSDT <= 0.0 && amountAlph <= 0.0 {
		fmt.Fprintf(os.Stderr, "Amount is mandatory\n")
		error = true
	}

	if amountUSDT > 0.0 && amountAlph > 0.0 {
		fmt.Fprintf(os.Stderr, "Cannot mix amount, select only one\n")
		error = true
	}

	if side == "" {
		fmt.Fprintf(os.Stderr, "Side is mandatory\n")
		error = true
	}

	if timeInForce != GOOD_TILL_CANCEL && timeInForce != IMMEDIATE_OR_CANCEL {
		fmt.Fprintf(os.Stderr, "Time in force accepted value. gtc or ioc\n")
		error = true
	}

	if error {
		flag.Usage()
		os.Exit(1)
	}

}
func getEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Some error occured. Err: %s", err)
	}

	error := false
	gateioKey = os.Getenv("GATEIO_KEY")
	if gateioKey == "" {
		fmt.Fprintf(os.Stderr, "Gate.io api key is missing\n")
		error = true

	}

	gateioSecret = os.Getenv("GATEIO_SECRET")
	if gateioSecret == "" {
		fmt.Fprintf(os.Stderr, "Gate.io secret key is missing\n")
		error = true

	}

	if error {
		flag.Usage()
		os.Exit(1)
	}

}

func balanceEnough(client *gateapi.APIClient, ctx *context.Context, currency string, amount float64) (bool, float64) {

	allBalances := checkBalance(client, ctx)

	for balanceIndex := 0; balanceIndex < len(allBalances); balanceIndex++ {
		if strings.ToUpper(allBalances[balanceIndex].Currency) == currency {

			available, err := strconv.ParseFloat(allBalances[balanceIndex].Available, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Formatting not possible for available balance: %s\n", err)
				panic(err)
			}

			locked, _ := strconv.ParseFloat(allBalances[balanceIndex].Locked, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Formatting not possible for locked balance: %s\n", err)
				panic(err)
			}

			if available-locked >= amount {
				return true, available - locked
			}
			return false, available - locked
		}
	}

	return false, 0.0

}

func printOpenOrders(client *gateapi.APIClient, ctx *context.Context) {
	openOrders := getOpenOrders(client, ctx, "ALPH_USDT")

	var buyOrders []gateapi.Order
	var sellOrders []gateapi.Order

	for i := 0; i < len(openOrders); i++ {
		if openOrders[i].Side == buy {
			buyOrders = append(buyOrders, openOrders[i])
		} else if openOrders[i].Side == sell {
			sellOrders = append(sellOrders, openOrders[i])
		}
	}

	amountFiat := 0.0
	amountCrypto := 0.0
	fmt.Println("Buy open orders")
	for i := 0; i < len(buyOrders); i++ {
		order := buyOrders[i]

		amount, err := strconv.ParseFloat(order.Amount, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Formatting not possible for formatOpenOrders: %s\n", err)
			panic(err)
		}

		price, err := strconv.ParseFloat(order.Price, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Formatting not possible for formatOpenOrders: %s\n", err)
			panic(err)
		}
		amountFiat += price * amount
		amountCrypto += amount
		formatOpenOrders(&order)
	}
	fmt.Printf("Total: +%.3f ALPH | -%.3f USDT\n", amountCrypto, amountFiat)

	amountFiat = 0.0
	amountCrypto = 0.0
	fmt.Println("Sell open orders")
	for i := 0; i < len(sellOrders); i++ {

		order := sellOrders[i]

		amount, err := strconv.ParseFloat(order.Amount, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Formatting not possible for formatOpenOrders: %s\n", err)
			panic(err)
		}

		price, err := strconv.ParseFloat(order.Price, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Formatting not possible for formatOpenOrders: %s\n", err)
			panic(err)
		}

		formatOpenOrders(&order)

		amountFiat += price * amount
		amountCrypto += amount

	}
	fmt.Printf("Total: -%.3f ALPH | +%.3f USDT", amountCrypto, amountFiat)

}

func printFilledOrders(client *gateapi.APIClient, ctx *context.Context, side string, limit int32) {
	var options gateapi.ListOrdersOpts

	if limit > 0 {
		//side is only compatible with limit
		options = gateapi.ListOrdersOpts{
			Limit: optional.NewInt32(limit),
			Side:  optional.NewString(side),
		}
	}

	if lastDays > 0 {
		options = gateapi.ListOrdersOpts{
			Limit: optional.NewInt32(1000),
			To:    optional.NewInt64(time.Now().Unix()),
			From:  optional.NewInt64(time.Now().Unix() - int64(lastDays*86400)),
		}
	}

	var orders []gateapi.Order
	orders = getOrders(client, ctx, "ALPH_USDT", "finished", &options)

	orderFilledCounter := 0
	avgPrice := 0.0
	var priceArray []float64
	totalToken := 0.0
	totalFiat := 0.0

	fmt.Printf("List %d %s orders\n", len(orders), side)
	for orderIndex := 0; orderIndex < len(orders); orderIndex++ {
		order := orders[orderIndex]

		filledTotal, _ := strconv.ParseFloat(order.FilledTotal, 64)
		if order.AvgDealPrice != "" && filledTotal > 0.0 && order.Side == side {

			orderFilledCounter++
			orderAvgPrice, _ := strconv.ParseFloat(order.AvgDealPrice, 64)

			orderAmount, _ := strconv.ParseFloat(order.Amount, 64)

			avgPrice += orderAvgPrice
			totalToken += orderAmount
			totalFiat += filledTotal
			priceArray = append(priceArray, orderAvgPrice)
			fmt.Printf("avg filled price: %s USDT, %.4f USDT, Volume: %.4f ALPH (created at: %s)\n", order.AvgDealPrice, filledTotal, orderAmount, time.Unix(order.CreateTimeMs/1000, 0))
		}
	}

	if len(orders) > 0 {
		operatorAlph := "-"
		operatorUsdt := "+"
		if side == buy {
			operatorAlph = "+"
			operatorUsdt = "-"
		}
		fmt.Printf("Avg paid price: %.4f USDT, median: %.4f USDT, Total: %s%.4f ALPH, Total: %s%.4f USDT\n\n", avgPrice/float64(orderFilledCounter), median(priceArray), operatorAlph, totalToken, operatorUsdt, totalFiat)
	}

}

func main() {

	getParams()
	getEnv()

	client := gateapi.NewAPIClient(gateapi.NewConfiguration())
	// uncomment the next line if your are testing against testnet
	// client.ChangeBasePath("https://fx-api-testnet.gateio.ws/api/v4")
	ctx := context.WithValue(context.Background(),
		gateapi.ContextGateAPIV4,
		gateapi.GateAPIV4{
			Key:    gateioKey,
			Secret: gateioSecret,
		},
	)
	// check if connected correctly
	getAccountDetails(client, &ctx)

	if listPastOrders {
		printFilledOrders(client, &ctx, buy, int32(limit))
		printFilledOrders(client, &ctx, sell, int32(limit))
		os.Exit(0)
	}
	if listOpenOrders {
		printOpenOrders(client, &ctx)
		os.Exit(0)
	}

	currentPrice := getTickerPrice(client, &ctx, "ALPH_USDT")
	useTriggeredOrder := false

	if (side == buy && priceMin >= currentPrice) || (side == sell && priceMin <= currentPrice) && !useSl {

		fmt.Printf("\nActual price is %.4f USDT, your %s orders will start at %.4f.\nIf you continue, the orders are going to be filled immediately\n1) Continue\n2) Use Stop-Limit orders\n3) Cancel\nChoice: ", currentPrice, side, priceMin)

		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		choice := strings.ToLower(input.Text())

		if choice == "1" {
			useTriggeredOrder = false
		} else if choice == "2" {
			useTriggeredOrder = true
		} else if choice == "3" {
			os.Exit(0)
		} else {
			os.Exit(0)
		}
		fmt.Println()

	}
	useTriggeredOrder = useSl

	if checkOrdersOpen(client, &ctx, "ALPH_USDT") {
		fmt.Printf("Some orders are already open\nDo you want to continue? [y/N] ")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()

		if strings.ToLower(input.Text()) != "y" {
			os.Exit(0)
		}
		fmt.Println()

	}

	fmt.Printf("Here are the orders you gonna create\n")
	var orders []gateapi.Order
	var sLOrders []gateapi.SpotPriceTriggeredOrder
	var ticker string  // specify the ticker we are going to use to buy or sell
	var amount float64 // amount in order

	if amountUSDT > 0.0 {
		ticker = "USDT"
		amount = amountUSDT
	} else if amountAlph > 0.0 {
		ticker = "ALPH"
		amount = amountAlph
	}

	balanceOk, balance := balanceEnough(client, &ctx, ticker, amount)
	if !balanceOk {
		fmt.Fprintf(os.Stderr, "\nNot enough %s, actual balance: %2.f needed: %.2f\n", ticker, balance, amount)
		os.Exit(1)
	}

	if !useTriggeredOrder {
		fmt.Printf("Using limit orders\n")
		orders = selectFiatOrCrypto(ticker, "ALPH_USDT", side, priceMin, priceMax, amount, steps, timeInForce)
	} else {
		fmt.Printf("Using Stop-limit orders\n")
		sLOrders = selectFiatOrCryptoTriggered(ticker, "ALPH_USDT", side, priceMin, priceMax, amount, steps)
	}

	fmt.Printf("\nDo you want to continue? [y/N] ")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()

	if strings.ToLower(input.Text()) != "y" {
		os.Exit(0)
	}

	fmt.Printf("\n")

	if !useTriggeredOrder {
		for i := 0; i < len(orders); i += GATE_MAX_SIZE_BATCH {
			end := i + GATE_MAX_SIZE_BATCH
			if end > len(orders) {
				end = len(orders)
			}

			chunk := orders[i:end]
			//fmt.Printf("Chunk %d: %+v\n", i/GATE_MAX_SIZE_BATCH+1, chunk)
			sendBatchOrder(client, &ctx, chunk)
			fmt.Printf("%d orders has been set", len(orders))
		}
	} else {

		for triggeredOrderIndex := 0; triggeredOrderIndex < len(sLOrders); triggeredOrderIndex++ {
			sendTriggeredOrder(client, &ctx, &sLOrders[triggeredOrderIndex])
		}
	}

}
