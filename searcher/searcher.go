package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/yugaraxy/searchutility"
	"github.com/yugaraxy/searchutility/crowler"
	"github.com/yugaraxy/searchutility/gauth"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

var (
	mu         sync.RWMutex
	valueRange *sheets.ValueRange

	srv       *sheets.Service
	readRange string

	startRow = flag.Int("start", 8, "search words started from Nth row in spreadsheet")
	endRow   = flag.Int("end", 10, "search words ended by Nth row in spreadsheet")
)

type GSEResponse struct {
	Items []Item `json:"items"`
}

type Item struct {
	SearchWords []string
	Rank        int
	Link        string `json:"link"`
	Title       string `json:"title"`
	Domain      string `json:"displayLink"`
	Description string `json:"snippet"`
}

func newGSEResponse(words string) *GSEResponse {
	ws := stringSlicer(words)
	r := new(GSEResponse)

	items := make([]Item, 10)
	for i, _ := range items {
		items[i].Rank = i + 1
		items[i].SearchWords = ws
	}
	r.Items = items
	return r
}

func (r GSEResponse) urlProcessor() {
	for _, item := range r.Items {
		crowler.CrowlAndShot(item.Link)
	}
}

func (r GSEResponse) Printer() {
	bytes, err := json.MarshalIndent(r, "", "	")
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	fmt.Println(string(bytes))
}

func wordsParser(words string) (result string) {
	s := strings.Split(words, " ")
	for i, w := range s {
		if i != 0 {
			result += "%20"
		}
		result += w
	}
	return result
}

func stringSlicer(words string) (result []string) {
	return strings.Split(words, ",")
}

func gSearcher(num, start int, words string, workers chan<- *GSEResponse, startTime time.Time) {
	num_of_results_param := "&num=" + strconv.Itoa(num)
	start_param := "&start=" + strconv.Itoa(start)
	lang_ja_param := "&lr=lang_ja"
	q_param := "&q=" + wordsParser(words)
	url := SEARCH_URL + "key=" + API_KEY +
		"&cx=" + SEARCH_ENGINE_ID +
		q_param + lang_ja_param +
		num_of_results_param + start_param

	fmt.Printf("%dms elapsed...", time.Now().Sub(startTime).Milliseconds())
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	fmt.Printf("get %s\n", url)
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("response status should be 200, but %d", resp.Status)
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	responds := newGSEResponse(words)
	err = json.NewDecoder(bytes.NewBuffer(content)).Decode(responds)
	if err != nil {
		panic(err)
	}

	for i, item := range responds.Items {
		item.Rank = i + 1
		item.SearchWords = stringSlicer(words)
		responds.Items[i] = item
	}
	workers <- responds
}

var cellFormatter string = `No.%d: %s
%s
%s
%s
`

func gWriter(workers <-chan *GSEResponse) {
	fmt.Println("gWriter started...")
	var cellCounter int

	mu.Lock()
	vlen := len(valueRange.Values)
	mu.Unlock()

	// update cells. insert values into a row for len(rows) times
	for i := 0; i < vlen; i++ {
		fmt.Println("\nwait workers in gWriter")
		r := <-workers
		for i, item := range r.Items {
			input := fmt.Sprintf(cellFormatter,
				item.Rank, item.Domain, item.Link, item.Title, item.Description)
			mu.Lock()
			if len(valueRange.Values[cellCounter]) > i+6 {
				valueRange.Values[cellCounter][i+6] = input
			} else {
				valueRange.Values[cellCounter] = append(valueRange.Values[cellCounter], input)
			}
			mu.Unlock()
		}
		cellCounter++
	}
	fmt.Println("get worker closing...")

	valueRange.Range = readRange
	caller := srv.Spreadsheets.Values.Update(SHEET_ID, readRange, valueRange)
	_, err := caller.ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		panic(err)
	}

	fmt.Println("succeeded")
}

func init() {
	flag.Parse()
	readRange = fmt.Sprintf("キーワード選定!B%d:Q%d", *startRow, *endRow)

	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := gauth.GetClient(config)

	srv, err = sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	valueRange, err = srv.Spreadsheets.Values.Get(SHEET_ID, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	if len(valueRange.Values) == 0 {
		panic("No data")
	}
}

func main() {
	workers := make(chan *GSEResponse, 5)
	startTime := time.Now()
	go func(t time.Time) {
		for _, row := range valueRange.Values {
			if row[0].(string) == "" { // skip empty row
				continue
			}
			// row[0]:  B (search words)
			// row[15]: Q
			gSearcher(10, 1, row[0].(string), workers, t)
		}
		close(workers)
	}(startTime)

	gWriter(workers)
}
