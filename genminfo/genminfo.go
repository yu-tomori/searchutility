// genminfo get media infomation from spreadsheet,
// and save in local json file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	. "github.com/yugaraxy/searchutility"
	"github.com/yugaraxy/searchutility/gauth"
	"github.com/yugaraxy/searchutility/mediainfo"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

var (
	valueRange *sheets.ValueRange

	srv       *sheets.Service
	readRange string

	startRow = flag.Int("start", 8, "read media info from Nth row in spreadsheet")
	endRow   = flag.Int("end", 10, "read media info ended by Nth row in spreadsheet")
)

func valueRangeConvert(vr *sheets.ValueRange) map[string]mediainfo.MediaInfo {
	mimap := make(map[string]mediainfo.MediaInfo, *endRow-*startRow+1)

	// row[0]		row[1]		row[2]		row[3]
	// ドメイン		メディア名	DR			UU
	for _, row := range vr.Values {
		var dr, uu float64

		if row[2].(string) != "" {
			var err error
			dr, err = strconv.ParseFloat(row[2].(string), 64)
			if err != nil {
				dr = float64(0)
			}
		}

		if row[3].(string) != "" {
			var err error
			uu, err = strconv.ParseFloat(row[3].(string), 64)
			if err != nil {
				uu = float64(0)
			}
		}

		mimap[row[0].(string)] = mediainfo.MediaInfo{
			Name:       row[1].(string),
			DomainRank: dr,
			UniqueUser: uu,
		}
	}

	return mimap
}

func init() {
	fmt.Println("initializing...")
	log.SetOutput(os.Stdout)
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
	fmt.Println("executed...")
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

	b, err := json.MarshalIndent(valueRangeConvert(valueRange), "", "\t")
	if err != nil {
		panic(err)
	}

	n, err := file.Write(b)
	if err != nil || n == 0 {
		panic(err)
	}
}
