package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/spf13/viper"
)

var baseUrl string
var token string

func main() {
	readConfig()

	url := baseUrl + "/api/bookmarks/"
	bearer := "Token " + token

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", bearer)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	var bookmarks1 *Bookmarks
	jsonErr := json.Unmarshal(body, &bookmarks1)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	for i := 0; i < len(bookmarks1.Results); i++ {
		testBookmark(&bookmarks1.Results[i])
	}
}

func readConfig() {
	viper.SetConfigName("config")                 // name of config file (without extension)
	viper.SetConfigType("yaml")                   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("$HOME/.linkdig-cleaner") // call multiple times to add many search paths
	viper.AddConfigPath(".")                      // optionally look for config in the working directory
	err := viper.ReadInConfig()                   // Find and read the config file
	if err != nil {                               // Handle errors reading the config file
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(fmt.Errorf("Error: %w \n", err))
		} else {
			panic(fmt.Errorf("Fatal error config file: %w \n", err))
		}
	}

	token = viper.Get("api_token").(string)
	baseUrl = viper.Get("base_url").(string)
}

/*
 * Makes a request to the bookmark URL and checks HTTP status codes.
 * If a 404 is found, it delegates archiving the link.
 */
func testBookmark(bmark *Bookmark) {
	var resp *http.Response
	var err error
	resp, err = http.Get(bmark.Url)
	if err != nil {
		print(err)
	}

	defer resp.Body.Close()

	renderStatusLine(resp.StatusCode, bmark.Url)

	if resp.StatusCode == http.StatusNotFound {
		archiveBookmark(bmark)
	}

	if resp.StatusCode == http.StatusForbidden {
		fmt.Println("Page forbidden access with " + bmark.Url)

		// body, err := ioutil.ReadAll(resp.Body)
		// if err != nil {
		// 	print(err)
		// }

		// fmt.Print(string(body))
		// fmt.Println("body")
	}
}

/*
 * Archives a bookmark.
 */
func archiveBookmark(bmark *Bookmark) {
	fmt.Println("  " + bmark.Website_title)
	fmt.Println("  " + bmark.Url)
	fmt.Println("\n  Archiving this bookmark…")

	url := baseUrl + "/api/bookmarks/" + strconv.Itoa(bmark.Id) + "/archive/"
	bearer := "Token " + token

	req, err := http.NewRequest("POST", url, nil)
	req.Header.Add("Authorization", bearer)
	if err != nil {
		log.Println("Error: ", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	fmt.Println("  … done!")
}

func renderStatusLine(statusCode int, url string) {
	// https://golangbyexample.com/print-output-text-color-console/
	colorRed := "\033[31m"
	colorYellow := "\033[33m"
	colorReset := "\033[0m"

	if statusCode == http.StatusNotFound {
		fmt.Println(string(colorRed), "[", statusCode, "] ", url, string(colorReset))
	} else if statusCode == http.StatusForbidden {
		fmt.Println(string(colorYellow), "[", statusCode, "] ", url, string(colorReset))
	} else {
		fmt.Println(" [", statusCode, "] ", url)
	}
}
