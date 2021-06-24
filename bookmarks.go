package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type Bookmark struct {
	Id            int
	Url           string
	Title         string
	Description   string
	Website_title string
	Tag_names     []string
}

type Bookmarks struct {
	Count    int
	Next     string
	Previous string
	Results  []Bookmark
}

/*
 * Gets all bookmarks from API
 */
func getAllBookmarks() *Bookmarks {
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

	return bookmarks1
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
