package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

var assetsName string     //name of assets, unputted by user
var done chan interface{} //websocket stream finish control

//handler for reading JSON data from websocket connection and writing it to chan
func receiveHandler(connection *websocket.Conn, data chan []byte) {
	defer close(done) //if there no data from stream
	for {
		_, msg, err := connection.ReadMessage()
		if err != nil {
			log.Println("Error in receive:", err)
			return
		}
		data <- msg
	}
}

//converting JSON data from chan and printing it to http.ResponseWriter
func dataPrinter(data chan []byte, writer http.ResponseWriter) {
	for {
		//parsing JSON data from channel to var orders
		var orders map[string]json.RawMessage
		jsonErr := json.Unmarshal(<-data, &orders)
		if jsonErr != nil {
			log.Println("error in json.Unmarshal:", jsonErr)
		}

		//parsing bids string slice from JSON data
		var bidsStr = [][]string{}
		jsonErr = json.Unmarshal(orders["bids"], &bidsStr)
		if jsonErr != nil {
			log.Println("error in json.Unmarshal:", jsonErr)
		}

		//parsing asks string slice from JSON data
		var asksStr = [][]string{}
		jsonErr = json.Unmarshal(orders["asks"], &asksStr)
		if jsonErr != nil {
			log.Println("error in json.Unmarshal:", jsonErr)
		}

		//converting string slices (bidsStr and asksStr) to float slices (bids and asks) to make calculations of total sums later
		var err error
		var bids [][]float64 //slice for saving bids
		var asks [][]float64 //slice for saving asks
		var price float64    //temp var for converting price of an order from string type to float
		var amount float64   //temp var for converting ammount of an order from string type to float

		//saving only 15 last bids and ask (because they are the newest ones), so that the last data in string slice (var orders) becomes the first data in float slices (var bids and asks)
		for i := len(bidsStr) - 1; i > len(bidsStr)-16; i-- {
			if price, err = strconv.ParseFloat(bidsStr[i][0], 64); err != nil {
				log.Println("Error in parsing:", err)
			}
			if amount, err = strconv.ParseFloat(bidsStr[i][1], 64); err != nil {
				log.Println("Error in parsing:", err)
			}
			bids = append(bids, []float64{price, amount})
		}
		for i := len(asksStr) - 1; i > len(asksStr)-16; i-- {
			if price, err = strconv.ParseFloat(asksStr[i][0], 64); err != nil {
				log.Println("Error in parsing:", err)
			}
			if amount, err = strconv.ParseFloat(asksStr[i][1], 64); err != nil {
				log.Println("Error in parsing:", err)
			}
			asks = append(asks, []float64{price, amount})
		}

		//printing data in console in table form
		//printing headers of table
		fmt.Fprintln(writer, "\t\t\t\t\t\t", strings.ToUpper(assetsName))
		fmt.Fprintln(writer, "\t\t      BIDS", "\t\t\t\t\t         ASKS")
		fmt.Fprintln(writer, "           price     amount       total", "\t\t       price     amount       total")

		//vars for orders sums
		var bidsPriceSum, bidsAmountSum, bidsTotalSum, asksPriceSum, asksAmountSum, asksTotalSum float64

		//printing data line by line and calculate sums
		for i := 0; i < len(bids) || i < len(asks); i++ {
			fmt.Fprintf(writer, "     %12.5f %9.5f %12.5f \t\t %12.5f %9.5f %12.5f \n", bids[i][0], bids[i][1], bids[i][0]*bids[i][1], asks[i][0], asks[i][1], asks[i][0]*asks[i][1])
			bidsPriceSum += bids[i][0]
			bidsAmountSum += bids[i][1]
			bidsTotalSum += bids[i][0] * bids[i][1]
			asksPriceSum += asks[i][0]
			asksAmountSum += asks[i][1]
			asksTotalSum += asks[i][0] * asks[i][1]
		}

		//printing sums of orders
		fmt.Fprintln(writer, "")
		fmt.Fprintf(writer, "Sums %12.5f %9.5f %12.5f \t\t %12.5f %9.5f %12.5f \n", bidsPriceSum, bidsAmountSum, bidsTotalSum, asksPriceSum, asksAmountSum, asksTotalSum)
	}
}

//handler for server which opens a websocket stream, reads and prints from stream
func viewHandler(writer http.ResponseWriter, request *http.Request) {

	//just instruction in console for user
	fmt.Println("Press Ctrl+c, if you want to stop server")

	interrupt := make(chan os.Signal, 1) //channel for interrupt signal
	signal.Notify(interrupt, os.Interrupt)
	data := make(chan []byte) //channel for data from websocket stream

	//open connection to websocket stream
	conn, _, err := websocket.DefaultDialer.Dial("wss://stream.binance.com:9443/ws/"+strings.ToLower(assetsName)+"@depth20@100ms", nil)
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()

	//start goroutines for reading and printing data from websocket stream
	go receiveHandler(conn, data)
	go dataPrinter(data, writer)

	//infinte loop for goroutines control
	for {
		select {

		case <-done: //end of stream
			return

		case <-interrupt: //ctrl+c
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Normal close by interrupt:", err)
				return
			}
		}
	}
}

func main() {
	//inputting assets name for retrieving orders book
	fmt.Println("For which assets do you want to get orders? (e.g. BNBBTC, BTCUSDT...)")
	fmt.Scanf("%s\n", &assetsName)

	fmt.Println("Connect to localhost:8080 using your browser, to start getting orders data from binance.com")

	// //setting handler for server
	http.HandleFunc("/", viewHandler)

	// //start local server
	log.Fatal(http.ListenAndServe(":8080", nil))
}
