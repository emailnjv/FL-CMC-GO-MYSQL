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

		for _, resultz := range results {
			defer wg1.Done()

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

			pairMatcher := func(n *html.Node) bool {
				if n.Parent.Parent.DataAtom == atom.Tr && n != nil && n.DataAtom == atom.A {
					return scrape.Attr(n, "class") != "market-name"
				}
				return false
			}
			var scrapedPair string
			var pairResult, _ = scrape.Find(resultz, pairMatcher)
			scrapedPair = scrape.Text(pairResult)

			/*
				-----------------------------------------------------------------------------------
			*/

			volumeMatcher := func(n *html.Node) bool {
				if n.Parent.DataAtom == atom.Td && n != nil && scrape.Attr(n, "class") == "volume" {
					return n.DataAtom == atom.Span
				}
				return false
			}
			var finalVolume float64
			var scrapedVolume string
			var volumeResult, _ = scrape.Find(resultz, volumeMatcher)
			scrapedVolume = scrape.Text(volumeResult)
			if scrapedVolume != "?" {
				withoutDollarSign := scrapedVolume[1:]
				withoutCommas := strings.Replace(withoutDollarSign, ",", "", -1)
				var parsedVolume, volumeParseError = strconv.ParseFloat(withoutCommas, 64)
				if volumeParseError != nil {
					fmt.Println("volumeParseError")
				}
				finalVolume = parsedVolume

			}

			/*
				-----------------------------------------------------------------------------------
			*/

			volumePercentMatcher := func(n *html.Node) bool {
				if n.Parent.Parent.DataAtom == atom.Tr && n != nil && n.DataAtom == atom.Span {
					return scrape.Attr(n, "data-format-value") != ""
				}
				return false
			}
			var scrapedVolumePercent string
			var volumePercentResult, _ = scrape.Find(resultz, volumePercentMatcher)
			scrapedVolumePercent = scrape.Attr(volumePercentResult, "data-format-value")
			var parsedVolumePercent, volumePercentParseError = strconv.ParseFloat(scrapedVolumePercent, 64)
			if volumePercentParseError != nil {
				fmt.Println("volumePercentParseError")
				fmt.Println(volumePercentParseError)
			}

			/*
				-----------------------------------------------------------------------------------
			*/

			updatedMatcher := func(n *html.Node) bool {
				if n.Parent.Parent.DataAtom == atom.Tbody && n != nil && scrape.Text(n) == "Recently" {
					return n.DataAtom == atom.Td
				}
				return false
			}
			var scrapedUpdated bool
			var _, updateError = scrape.Find(resultz, updatedMatcher)
			if updateError {
				scrapedUpdated = true
			}

			/*
				-----------------------------------------------------------------------------------
			*/

			InsertExchange(Exchange{exchangeTitle, scrapedName, scrapedPair, finalVolume, parsedPrice, parsedVolumePercent, scrapedUpdated})

		}
		go func() {
			wg1.Wait()
			wg.Done()
		}()

	}
	go func() {
		wg.Wait()
		fmt.Println("DONE")
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

			/*
				-----------------------------------------------------------------------------------
			*/

			priceMatcher := func(n *html.Node) bool {
				if n.Parent.Parent.DataAtom == atom.Tr && n != nil && n.DataAtom == atom.A {
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
					return scrape.Attr(n, "class") == "currency-name-container"
				}
				return false
			}
			var scrapedName string
			var nameResult, _ = scrape.Find(resultz, nameMatcher)
			scrapedName = scrape.Text(nameResult)

			/*
				-----------------------------------------------------------------------------------
			*/

			// symbolMatcher := func(n *html.Node) bool {
			// 	if n.Parent.DataAtom == atom.Tr && n != nil && n.DataAtom == atom.Td {
			// 		return scrape.Attr(n, "class") != "col-symbol"
			// 	}
			// 	return false
			// }
			var scrapedSymbol string
			var symbolResult, _ = scrape.Find(resultz, scrape.ByClass("col-symbol"))
			scrapedSymbol = scrape.Text(symbolResult)

			/*
				-----------------------------------------------------------------------------------
			*/

			// marketCapMatcher := func(n *html.Node) bool {
			// 	if n.Parent.DataAtom == atom.Tr && n != nil && scrape.Attr(n, "class") == "market-cap" {
			// 		if n.DataAtom == atom.Td {
			// 			return true
			// 		}
			// 	}
			// 	return false
			// }
			var finalMarketCap float64
			var scrapedMarketCap string
			var marketCapResult, _ = scrape.Find(resultz, scrape.ByClass("market-cap"))
			scrapedMarketCap = scrape.Attr(marketCapResult, "data-usd")
			if scrapedMarketCap != "?" {
				// withoutDollarSign := scrapedVolume[1:]
				// withoutCommas := strings.Replace(withoutDollarSign, ",", "", -1)
				var parsedMarketCap, parsedMarketCapError = strconv.ParseFloat(scrapedMarketCap, 64)
				if parsedMarketCapError != nil {
					fmt.Println("volumeParseError")
				}
				finalMarketCap = parsedMarketCap

			}

			/*
				-----------------------------------------------------------------------------------
			*/

			// volumePercentMatcher := func(n *html.Node) bool {
			// 	if n.Parent.Parent.DataAtom == atom.Tr && n != nil && n.DataAtom == atom.Span {
			// 		return scrape.Attr(n, "data-format-value") != ""
			// 	}
			// 	return false
			// }
			// var scrapedVolumePercent string
			// var volumePercentResult, _ = scrape.Find(resultz, volumePercentMatcher)
			// scrapedVolumePercent = scrape.Attr(volumePercentResult, "data-format-value")
			// var parsedVolumePercent, volumePercentParseError = strconv.ParseFloat(scrapedVolumePercent, 64)
			// if volumePercentParseError != nil {
			// 	fmt.Println("volumePercentParseError")
			// 	fmt.Println(volumePercentParseError)
			// }

			/*
				-----------------------------------------------------------------------------------
			*/

			// circulatingSupplyMatcher := func(n *html.Node) bool {
			// 	if n.Parent.Parent.DataAtom == atom.Tr && n != nil && n.DataAtom == atom.Span {
			// 		return scrape.Attr(n, "data-format-value") != ""
			// 	}
			// 	return false
			// }
			// var scrapedCirculatingSupply string
			// var circulatingSupplyResult, _ = scrape.Find(resultz, circulatingSupplyMatcher)
			// scrapedCirculatingSupply = scrape.Attr(volumePercentResult, "data-format-value")
			// var parsedVolumePercent, volumePercentParseError = strconv.ParseFloat(scrapedVolumePercent, 64)
			// if volumePercentParseError != nil {
			// 	fmt.Println("volumePercentParseError")
			// 	fmt.Println(volumePercentParseError)
			// }

			/*
				-----------------------------------------------------------------------------------
			*/

			// updatedMatcher := func(n *html.Node) bool {
			// 	if n.Parent.Parent.DataAtom == atom.Tbody && n != nil && scrape.Text(n) == "Recently" {
			// 		return n.DataAtom == atom.Td
			// 	}
			// 	return false
			// }
			// var scrapedUpdated bool
			// var _, updateError = scrape.Find(resultz, updatedMatcher)
			// if updateError {
			// 	scrapedUpdated = true
			// }

			/*
				-----------------------------------------------------------------------------------
			*/

			fmt.Println(scrapedName)
			fmt.Println(scrapedSymbol)
			fmt.Println(finalMarketCap)
			fmt.Println(parsedPrice)

			// InsertCurrency(Currency{scrapedName, scrapedSymbol, finalMarketCap, parsedPrice, parsupply, volS, oned, oneh, onew})

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

	// resultNodeExchangeGen(rootGen(respGen(urlArr.Urls...)))
	resultNodeCurrencyGen(rootGen(respGen("https://coinmarketcap.com/all/views/all/")))
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
