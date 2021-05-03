package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
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

func main() {
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
	responds := new(GSEResponse)
	err = json.NewDecoder(bytes.NewBuffer(content)).Decode(responds)
	if err != nil {
		panic(err)
	}
	responds.Printer()
}
