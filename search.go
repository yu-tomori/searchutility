package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/sclevine/agouti"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var num = flag.String("n", "10", "検索結果の数")
var start = flag.String("s", "1", "指定されたランクから検索")
var words = flag.String("w", "shopify,最高", "検索ワードを,区切りで指定。")

var (
	SEARCH_ENGINE_ID string
	API_KEY          string
	SEARCH_URL       string
)

type GSEResponse struct {
	Items []Item `json:"items"`
}

type Item struct {
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
	s := strings.Split(words, ",")
	for i, w := range s {
		if i != 0 {
			result += "%20"
		}
		result += w
	}
	return result
}

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

func init() {
	flag.Parse()
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
}

func main() {
	num_of_results_param := "&num=" + *num
	start_param := "&start=" + *start
	lang_ja_param := "&lr=lang_ja"
	q_param := "&q=" + wordsParser(*words)
	url := SEARCH_URL + "key=" + API_KEY + "&cx=" + SEARCH_ENGINE_ID + q_param + lang_ja_param + num_of_results_param + start_param
	fmt.Printf("URL:\n%s\n\n", url)
	resp, err := http.Get(url)
	log.Printf("url: %s", url)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("response status should be 200, but %d", resp.Status)
		os.Exit(0)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(content))
	responds := new(GSEResponse)
	err = json.NewDecoder(bytes.NewBuffer(content)).Decode(responds)
	if err != nil {
		panic(err)
	}
	responds.Printer()
	responds.urlProcessor()
}
