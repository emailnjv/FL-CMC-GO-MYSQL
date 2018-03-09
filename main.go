package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

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
	Pair              string
	Volume            int
	Price             int
	CirculatingSupply int
	oneHour           int
	oneDay            int
	oneWeek           int
}

type Exchange struct {
	Currency        string
	Pair            string
	VolumeCur       int
	Price           int
	VolumePercent   int
	UpdatedRecently int
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
	insForm, err := db.Prepare("INSERT INTO exchange(TimeScraped=NOW(), Currency=?, Pair=?, VolumeCur=?, Price=?, VolumePercent=?, UpdatedRecently=?)")
	if err != nil {
		panic(err.Error())
	}
	insForm.Exec(Currency, Pair, VolumeCur, Price, VolumePercent, UpdatedRecently)
	log.Println(Currency, Pair, VolumeCur, Price, VolumePercent, UpdatedRecently)

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

		var results = scrape.FindAll(root, rowMatcher)

		var wg1 sync.WaitGroup
		wg1.Add(len(results))
		// ********
		for _, result := range results {

			fmt.Println(scrape.Text(result))
			out := make(chan *html.Node)
			go func(n *html.Node) {
				defer wg1.Done()
				if strings.Contains(scrape.Text(n), "Sponsored P.when") {
					preString := strings.SplitAfter(scrape.Text(n), "? Leave ad feedback ")
					out <- preString[1]
				} else {
					out <- scrape.Text(n)
				}
			}(result)
		}
		go func() {
			wg1.Wait()
			wg.Done()
		}()

	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func getUrls() {
	urlArr := new(UrlArrStruct)
	raw, err := ioutil.ReadFile("./sites.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	json.Unmarshal(raw, &urlArr)

	for result := range resultNodeExchangeGen(rootGen(respGen(urlArr.Urls...))) {
		fmt.Println(result)
	}
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
