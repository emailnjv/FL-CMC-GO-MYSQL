package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (

// resultList [] AmazonResult

)

type Currency struct {
	Name              string
	Symbol            string
	MarketCap         float64
	Price             float64
	CirculatingSupply float64
	Volume            float64
	oneHour           float64
	oneDay            float64
	oneWeek           float64
}

type Exchange struct {
	ExchangeName    string
	Currency        string
	Pair            string
	VolumeCur       float64
	Price           float64
	VolumePercent   float64
	UpdatedRecently bool
}

type UrlArrStruct struct {
	Urls []string `json:"urls"`
}

func dbConn() (db *sql.DB) {
	dbDriver := "mysql"
	dbUser := "root"
	dbPass := "toor"
	dbName := "coin_market_cap"
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	return db
}

func InsertExchange(exch Exchange) {
	db := dbConn()
	Currency := exch.Currency
	Pair := exch.Pair
	VolumeCur := exch.VolumeCur
	Price := exch.Price
	VolumePercent := exch.VolumePercent
	UpdatedRecently := exch.UpdatedRecently
	insForm, err := db.Prepare("INSERT INTO exchanges(TimeScraped, ExchangeTitle, Currency, Pair, VolumeCur, Price, VolumePercent, UpdatedRecently) VALUES(?,?,?,?,?,?,?,?)")
	if err != nil {
		panic(err.Error())
	}
	insForm.Exec(time.Now(), exch.ExchangeName, Currency, Pair, VolumeCur, Price, VolumePercent, UpdatedRecently)

	defer db.Close()
}
func InsertCurrency(curr Currency) {
	db := dbConn()
	name := curr.Name
	symbol := curr.Symbol
	VolumeCur := curr.Volume
	Price := curr.Price
	CirculatingSupply := curr.CirculatingSupply
	oneHour := curr.oneHour
	oneDay := curr.oneDay
	oneWeek := curr.oneWeek

	insForm, err := db.Prepare("INSERT INTO all_currencies(TimeScraped, Currency, Symbol, Volume, Price, CirculatingSupply, oneHour, oneDay,oneWeek) VALUES(?,?,?,?,?,?,?,?,?)")
	if err != nil {
		panic(err.Error())
	}
	insForm.Exec(time.Now(), name, symbol, VolumeCur, Price, CirculatingSupply, oneHour, oneDay, oneWeek)

	defer db.Close()
}

func respGen(urls ...string) <-chan *http.Response {
	var wg sync.WaitGroup
	out := make(chan *http.Response)
	wg.Add(len(urls))
	for _, url := range urls {
		go func(url string) {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				panic(err)
			}
			req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				panic(err)
			}
			out <- resp
			wg.Done()
		}(url)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func rootGen(in <-chan *http.Response) <-chan *html.Node {
	var wg sync.WaitGroup
	out := make(chan *html.Node)
	for resp := range in {
		wg.Add(1)
		go func(resp *http.Response) {
			root, err := html.Parse(resp.Body)
			if err != nil {
				panic(err)
			}
			out <- root
			wg.Done()
		}(resp)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func resultNodeExchangeGen(in <-chan *html.Node) bool {
	var wg sync.WaitGroup
	for root := range in {
		wg.Add(1)

		rowMatcher := func(n *html.Node) bool {
			if n.DataAtom == atom.Tr && n != nil && scrape.Attr(n.Parent.Parent, "id") == "exchange-markets" {
				return n.Parent.DataAtom == atom.Tbody
			}
			return false
		}

		exchangeMatcher := func(n *html.Node) bool {
			if n.DataAtom == atom.H1 {
				return true
			}
			return false
		}

		var results = scrape.FindAll(root, rowMatcher)
		var headResults, _ = scrape.Find(root, exchangeMatcher)
		exchangeTitle := scrape.Text(headResults)

		var wg1 sync.WaitGroup
		wg1.Add(len(results))
		// ********

		for _, resultz := range results {
			defer wg1.Done()

			arr := strings.Split(scrape.Text(resultz), " ")

			var updateMatcherResult bool
			if arr[len(arr)-1] == "Recently" {
				updateMatcherResult = true

			}
			if arr[len(arr)-1] != "Recently" {
				updateMatcherResult = false
				arr = arr[:len(arr)-2]

			}

			var arrr1 string
			if arr[len(arr)-5] != "***" && arr[len(arr)-5] != "*" {
				arrr1 = arr[len(arr)-5][1:]
				if arrr1 == "***" {
					arrr1 = arr[len(arr)-6][1:]
				}
			}
			if arr[len(arr)-5] == "***" {
				arrr1 = arr[len(arr)-4][1:]
				if arrr1 == "***" {
					arrr1 = arr[len(arr)-6][1:]
				}
			}
			if arr[len(arr)-5] == "*" {
				arrr1 = arr[len(arr)-4][1:]
				if arrr1 == "***" {
					arrr1 = arr[len(arr)-6][1:]
				}
			}

			arrr12 := strings.Replace(arrr1, ",", "", -1)
			var arrr13, err1 = strconv.ParseFloat(arrr12, 64)
			if err1 != nil {
				fmt.Println(err1, arrr13)
			}



			arr3 := arr[len(arr)-3]
			arr32 := strings.Replace(arr3, ",", "", -1)
			var arrr33, err3 = strconv.ParseFloat(arr32, 64)
			if err3 != nil {
				fmt.Println(err3)
				fmt.Println("&&&&&&&&&&&&&&&&")
			}




			/*
			-----------------------------------------------------------------------------------
			*/



			priceMatcher := func(n *html.Node) bool {
				if n.Parent.Parent.DataAtom == atom.Tr && n != nil && n.DataAtom == atom.Span {
					return scrape.Attr(n, "class") == "price"
				}
				return false
			}
			var scrapedPrice string
			var priceResult, _ = scrape.Find(resultz, priceMatcher)
			scrapedPrice = scrape.Attr(priceResult, "data-usd")
			var parsedPrice, priceParseError = strconv.ParseFloat(scrapedPrice, 64)
			if priceParseError != nil {
				fmt.Println("priceParseError")
				fmt.Println(priceParseError)
			}



			/*
			-----------------------------------------------------------------------------------
			*/


			nameMatcher := func(n *html.Node) bool {
				if n.Parent.Parent.DataAtom == atom.Tr && n != nil && n.DataAtom == atom.A {
					return scrape.Attr(n, "class") == "market-name"
				}
				return false
			}
			var scrapedName string
			var nameResult, _ = scrape.Find(resultz, nameMatcher)
			scrapedName = scrape.Text(nameResult)



			/*
			-----------------------------------------------------------------------------------
			*/

			fmt.Printf(scrapedName)

			InsertExchange(Exchange{exchangeTitle, scrapedName, arr[len(arr)-6], arrr13, parsedPrice, arrr33, updateMatcherResult})

		}
		go func() {
			wg1.Wait()
			wg.Done()
		}()

	}
	go func() {
		wg.Wait()
	}()
	return true

}
func resultNodeCurrencyGen(in <-chan *html.Node) bool {
	var wg sync.WaitGroup
	for root := range in {
		wg.Add(1)

		rowMatcherQ := func(n *html.Node) bool {
			if n.DataAtom == atom.Tr && n != nil && scrape.Attr(n.Parent.Parent, "id") == "currencies-all" {
				return n.Parent.DataAtom == atom.Tbody
			}
			return false
		}

		var results = scrape.FindAll(root, rowMatcherQ)

		var wg1 sync.WaitGroup
		wg1.Add(len(results))
		// ********
		for _, resultz := range results {
			defer wg1.Done()

			arr := strings.Split(scrape.Text(resultz), " ")
			fmt.Println(arr)

			currSymbol := arr[1]

			// arr3 := arr[len(arr)-3][1:]
			// arr32 := strings.Replace(arr3, ",", "", -1)
			// var arrr33, err3 = strconv.ParseFloat(arr32, 64)
			// if err3 != nil {
			// 	fmt.Println(err3)
			// }
			var newArr []string
			if arr[len(arr)-5] == "*" {
				var slice1 []string
				slice1 = arr[:len(arr)-6]
				var slice2 []string
				slice2 = arr[len(arr)-4:]
				slice1 = append(slice1, slice2...)
				newArr = slice1
			}
			// modifiedNameString := strings.Replace(arr, ",", "", -1)
			var currName string
			currName = strings.Join(newArr[1:len(newArr)-8], " ")

			marketc := arr[len(arr)-7][1:]
			marketcWTH := strings.Replace(marketc, ",", "", -1)
			var parsedMC, err2 = strconv.ParseFloat(marketcWTH, 64)
			if err2 != nil {
				fmt.Println(err2)
			}

			price := arr[len(arr)-6][1:]
			priceWTH := strings.Replace(price, ",", "", -1)
			var parsedPrice, err3 = strconv.ParseFloat(priceWTH, 64)
			if err3 != nil {
				fmt.Println(err3)
			}
			volume := arr[len(arr)-4][1:]
			volumeWTH := strings.Replace(volume, ",", "", -1)
			var volS, err34 = strconv.ParseFloat(volumeWTH, 64)
			if err34 != nil {
				fmt.Println(err34)
			}
			supply := arr[len(arr)-5]
			supplyWTH := strings.Replace(supply, ",", "", -1)
			var parsupply, err35 = strconv.ParseFloat(supplyWTH, 64)
			if err35 != nil {
				fmt.Println(err35)
			}
			hour := arr[len(arr)-3]
			hourWTH := strings.Replace(hour, "%", "", -1)
			var onehe, err325 = strconv.ParseFloat(hourWTH, 64)
			var oneh = onehe / 100
			if err325 != nil {
				fmt.Println(err325)
			}
			day := arr[len(arr)-2]
			dayWTH := strings.Replace(day, "%", "", -1)
			var onede, err3225 = strconv.ParseFloat(dayWTH, 64)
			var oned = onede / 100
			if err3225 != nil {
				fmt.Println(err3225)
			}
			week := arr[len(arr)-1]
			weekWTH := strings.Replace(week, "%", "", -1)
			var onewe, err1225 = strconv.ParseFloat(weekWTH, 64)
			var onew = onewe / 100
			if err1225 != nil {
				fmt.Println(err1225)
			}

			InsertCurrency(Currency{currName, currSymbol, parsedMC, parsedPrice, parsupply, volS, oned, oneh, onew})

		}
		go func() {
			wg1.Wait()
			wg.Done()
		}()

	}
	go func() {
		wg.Wait()
	}()
	return true

}

func getUrls() {
	urlArr := new(UrlArrStruct)
	raw, err := ioutil.ReadFile("./sites.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	json.Unmarshal(raw, &urlArr)

	resultNodeExchangeGen(rootGen(respGen(urlArr.Urls...)))
	// resultNodeCurrencyGen(rootGen(respGen("https://coinmarketcap.com/all/views/all/")))
	// jsonData, err := json.Marshal(resultList)

	if err != nil {
		panic(err)
	}

	// // write to JSON file
	// jsonFile, err := os.Create("./DellResults.json")

	// if err != nil {
	// 	panic(err)
	// }
	// defer jsonFile.Close()

	// jsonFile.Write(jsonData)
	// jsonFile.Close()
	// fmt.Println("JSON data written to ", jsonFile.Name())

}

func main() {
	getUrls()
}
