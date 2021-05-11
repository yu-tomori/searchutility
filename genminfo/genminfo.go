// genminfo get media infomation from spreadsheet,
// and save in local json file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	. "github.com/yugaraxy/searchutility"
	"github.com/yugaraxy/searchutility/gauth"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
	"io/ioutil"
	"log"
	"os"
)

var (
	valueRange *sheets.ValueRange

	srv       *sheets.Service
	readRange string

	startRow = flag.Int("start", 8, "read media info from Nth row in spreadsheet")
	endRow   = flag.Int("end", 10, "read media info ended by Nth row in spreadsheet")
)

func init() {
	flag.Parse()
	readRange = fmt.Sprintf("メディアの基本情報!E%d:H%d", *startRow, *endRow)

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
	f := "mediainfo.json"

	// if mediainfo.json exists, remove it.
	if _, err := os.Stat(f); err == nil {
		e := os.Remove(f)
		if e != nil {
			panic(err)
		}
	}

	file, err := os.Create(f)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	b, err := json.MarshalIndent(*valueRange, "", "\t")
	if err != nil {
		panic(err)
	}

	n, err := file.Write(b)
	if err != nil || n == 0 {
		panic(err)
	}
}
