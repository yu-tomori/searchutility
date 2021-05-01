package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	SEARCH_ENGINE_ID string
	API_KEY          string
	SEARCH_URL       string
)

func main() {
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

	url := SEARCH_URL + "key=" + API_KEY + "&cx=" + SEARCH_ENGINE_ID + "&q=lectures"
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
	fmt.Println(resp.ContentLength)
	body := make([]byte, int(resp.ContentLength))
	resp.Body.Read(body)
	fmt.Println(string(body))
}
