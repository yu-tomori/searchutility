package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/sclevine/agouti"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

var (
	SEARCH_ENGINE_ID string
	API_KEY          string
	SEARCH_URL       string

	mu         sync.RWMutex
	valueRange *sheets.ValueRange

	srv *sheets.Service
)

const (
	SHEET_ID   string = "1dpn2o9mKRCEZBh8avTn41d1BD9br-ryY6BZyfguDv4A"
	READ_RANGE string = "キーワード選定!B8:Q12"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func crowlAndShot(urlstring string) {
	agoutiDriver := agouti.ChromeDriver()
	agoutiDriver.Start()
	defer agoutiDriver.Stop()
	page, _ := agoutiDriver.NewPage()

	page.Navigate(urlstring)

	u, err := url.Parse(urlstring)
	if err != nil {
		panic(err)
	}

	filename := "./" + u.Host + ".png"
	if err := page.Screenshot(filename); err != nil {
		panic(err)
	}
}

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

func (r GSEResponse) urlProcessor() {
	for _, item := range r.Items {
		crowlAndShot(item.Link)
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

func gSearcher(num, start int, words string, workers chan<- *GSEResponse) {
	num_of_results_param := "&num=" + strconv.Itoa(num)
	start_param := "&start=" + strconv.Itoa(start)
	lang_ja_param := "&lr=lang_ja"
	q_param := "&q=" + wordsParser(words)
	url := SEARCH_URL + "key=" + API_KEY + "&cx=" + SEARCH_ENGINE_ID + q_param + lang_ja_param + num_of_results_param + start_param
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\nget %s\n", url)
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("response status should be 200, but %d", resp.Status)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	responds := new(GSEResponse)
	err = json.NewDecoder(bytes.NewBuffer(content)).Decode(responds)
	if err != nil {
		panic(err)
	}
	for i, item := range responds.Items {
		item.Rank = i + 1
		item.SearchWords = stringSlicer(words)
		responds.Items[i] = item
	}
	fmt.Println("\n-----gSearcher-----")
	fmt.Printf("r: %T, %v", *responds, *responds)
	fmt.Println("\n-----gSearcher-----")

	workers <- responds
}

func init() {
	file, err := os.Open(".env")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	/* load configuration */
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		key := strings.Split(line, "=")[0]
		value := strings.Split(line, "=")[1]
		switch key {
		case "SEARCH_ENGINE_ID":
			SEARCH_ENGINE_ID = value
		case "SEARCH_URL":
			SEARCH_URL = value
		case "API_KEY":
			API_KEY = value
		}
	}

	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err = sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	valueRange, err = srv.Spreadsheets.Values.Get(SHEET_ID, READ_RANGE).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	if len(valueRange.Values) == 0 {
		panic("No data")
	}
}

var cellCounter int = 0

func gWriter(workers <-chan *GSEResponse) {
	fmt.Println("\n=========================")
	fmt.Println("gWriter started...")
	for _ = range valueRange.Values {
		fmt.Println("\nwait workers in gWriter")
		r := <-workers
		fmt.Printf("\nr: %T, value: %v", r, r)
		fmt.Println("\ncatch GSEResponse in gWriter")
		for i, item := range r.Items {
			fmt.Println("\n--------------------------")
			fmt.Printf("Words: %v, Rank: %s\n", item.SearchWords, string(item.Rank))
			input := "Rank: " + string(item.Rank) + "\n" +
				"Domain: " + item.Domain + "\n" +
				"Link: " + item.Link + "\n" +
				"Title: " + item.Title + "\n" +
				"Description: " + item.Description + "\n"
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

	valueRange.Range = READ_RANGE
	caller := srv.Spreadsheets.Values.Update(SHEET_ID, READ_RANGE, valueRange)
	_, err := caller.ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		panic(err)
	}
	fmt.Println("succeeded")

	b, _ := valueRange.MarshalJSON()
	fmt.Println(string(b))
}

func main() {
	workers := make(chan *GSEResponse, 5)
	go func() {
		for _, row := range valueRange.Values {
			fmt.Printf("\nLen(row): %d\n", len(row))
			// row[0]:  B (search words)
			// row[15]: Q
			gSearcher(10, 1, row[0].(string), workers)
		}
		close(workers)
	}()
	gWriter(workers)
}
