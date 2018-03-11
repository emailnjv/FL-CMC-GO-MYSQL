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

type Currency struct {
	Id                string `json:"id"`
	Name              string `json:"name"`
	Symbol            string `json:"symbol"`
	Rank              string `json:"rank"`
	PriceUsd          string `json:"price_usd"`
	PriceBtc          string `json:"price_btc"`
	Twenty4hVolumeUsd string `json:"24h_volume_usd"`
	MarketCapUsd      string `json:"market_cap_usd"`
	AvailableSupply   string `json:"available_supply"`
	TotalSupply       string `json:"total_supply"`
	MaxSupply         string `json:"max_supply"`
	PercentChange1h   string `json:"percent_change_1h"`
	PercentChange24h  string `json:"percent_change_24h"`
	PercentChange7d   string `json:"percent_change_7d"`
	LastUpdated       string `json:"last_updated"`
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
	db, err := sql.Open("mysql", "root:toor@/coin_market_cap")
	if err != nil {
		fmt.Println(err)
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
	fmt.Println("Inserted: ", exch.ExchangeName)
	defer db.Close()
}
func InsertCurrency(curr Currency) {
	db := dbConn()
	name := curr.Name
	symbol := curr.Symbol
	marketCap := curr.MarketCapUsd
	price := curr.PriceUsd
	circulatingSupply := curr.TotalSupply
	volume := curr.Twenty4hVolumeUsd
	oneHour := curr.PercentChange1h
	oneDay := curr.PercentChange24h
	oneWeek := curr.PercentChange7d

	var parsedMarketCap float64
	var parsedMarketCapError error
	if marketCap != "" {
		parsedMarketCap, parsedMarketCapError = strconv.ParseFloat(marketCap, 64)
		if parsedMarketCapError != nil {
			fmt.Println("parsedMarketCapError")
		}
	}

	if marketCap == "" {
		fmt.Println(curr, "MARKET Data is VACANT from Coin Market Cap")
	}

	var parsedVolume float64
	var parsedVolumeError error
	if volume != "" {
		parsedVolume, parsedVolumeError = strconv.ParseFloat(volume, 64)
		if parsedVolumeError != nil {
			fmt.Println("parsedVolumeError")
		}
	}
	if volume == "" {
		fmt.Println(curr, "VOLUME Data is VACANT from Coin Market Cap")
	}

	var parsedPrice float64
	var parsedPriceError error
	if price != "" {
		parsedPrice, parsedPriceError = strconv.ParseFloat(price, 64)
		if parsedPriceError != nil {
			fmt.Println("parsedPriceError")
		}
	}
	if price == "" {
		parsedPrice, parsedPriceError = strconv.ParseFloat(price, 64)
		fmt.Println(curr, "PRICE Data is VACANT from Coin Market Cap")
	}

	var parsedCirculatingSupply float64
	var parsedCirculatingSupplyError error
	if circulatingSupply != "" {
		parsedCirculatingSupply, parsedCirculatingSupplyError = strconv.ParseFloat(circulatingSupply, 64)
		if parsedCirculatingSupplyError != nil {
			fmt.Println("parsedCirculatingSupplyError")
		}
	}
	if circulatingSupply == "" {
		fmt.Println(curr, "CIRCULATING SUPPLY Data is VACANT from Coin Market Cap")
	}

	var parsedOneHour float64
	var parsedOneHourError error
	if oneHour != "" {
		parsedOneHour, parsedOneHourError = strconv.ParseFloat(oneHour, 64)
		if parsedOneHourError != nil {
			fmt.Println("parsedOneHourError")
		}
	}
	if oneHour == "" {
		fmt.Println(curr, "ONE HOUR Data is VACANT from Coin Market Cap")
	}

	var parsedOneDay float64
	var parsedOneDayError error
	if oneDay != "" {
		parsedOneDay, parsedOneDayError = strconv.ParseFloat(oneDay, 64)
		if parsedOneDayError != nil {
			fmt.Println("parsedOneDayError")
		}
	}

	if oneDay == "" {
		fmt.Println(curr, "ONE DAY Data is VACANT from Coin Market Cap")
	}

	var parsedOneWeek float64
	var parsedOneWeekError error
	if oneWeek != "" {
		parsedOneWeek, parsedOneWeekError = strconv.ParseFloat(oneWeek, 64)
		if parsedOneWeekError != nil {
			fmt.Println("parsedOneWeekError")
		}
	}

	if oneWeek == "" {
		fmt.Println(curr, "ONE WEEK Data is VACANT from Coin Market Cap")
	}

	insForm, err := db.Prepare("INSERT INTO all_currencies(TimeScraped, Currency, Symbol, MarketCap, Volume, Price, CirculatingSupply, OneHour, OneDay,OneWeek) VALUES(?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		panic(err.Error())
	}
	defer func() {
		insForm.Exec(time.Now(), name, symbol, parsedMarketCap, parsedVolume, parsedPrice, parsedCirculatingSupply, parsedOneHour, parsedOneDay, parsedOneWeek)
		db.Close()
	}()

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
			time.Sleep(3 * time.Second)
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

		var wg2 sync.WaitGroup
		wg2.Add(len(results))
		// ********
		for _, resultz := range results {
			defer wg2.Done()

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
			wg2.Wait()
			wg.Done()
		}()

	}
	go func() {
		wg.Wait()
		fmt.Println("DONE")
	}()
	return true

}
func resultNodeCurrencyGen(in <-chan *http.Response) bool {
	//var wgCurrency sync.WaitGroup
	for apiResponse := range in {

		body, apirReadErr := ioutil.ReadAll(apiResponse.Body)
		if apirReadErr != nil {
			fmt.Println(apirReadErr.Error())
		}
		apiResponse := new([]Currency)
		jsonErr := json.Unmarshal(body, &apiResponse)
		if jsonErr != nil {
			fmt.Println("whoops:", jsonErr)
		}

		for _, indCurrency := range *apiResponse {
			if indCurrency.MarketCapUsd == "" {
				fmt.Println(indCurrency)
			}

			InsertCurrency(Currency{indCurrency.Id, indCurrency.Name, indCurrency.Symbol, indCurrency.Rank, indCurrency.PriceUsd, indCurrency.PriceBtc, indCurrency.Twenty4hVolumeUsd, indCurrency.MarketCapUsd, indCurrency.AvailableSupply, indCurrency.TotalSupply, indCurrency.MaxSupply, indCurrency.PercentChange1h, indCurrency.PercentChange24h, indCurrency.PercentChange7d, indCurrency.LastUpdated})

		}

	}

	return true

}

func getUrls() bool {
	urlArr := new(UrlArrStruct)
	raw, err := ioutil.ReadFile("./sites.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	json.Unmarshal(raw, &urlArr)

	resultNodeCurrencyGen(respGen("https://api.coinmarketcap.com/v1/ticker/?limit=0"))
	resultNodeExchangeGen(rootGen(respGen(urlArr.Urls...)))

	if err != nil {
		panic(err)
	}
	return true

}

func main() {
	fmt.Println(getUrls())
}
