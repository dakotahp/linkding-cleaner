package main

import (
	"fmt"
	"net/http"

	"github.com/spf13/viper"
)

var baseUrl string
var token string

func main() {
	readConfig()

	bookmarks := getAllBookmarks()

	for i := 0; i < len(bookmarks.Results); i++ {
		testBookmark(&bookmarks.Results[i])
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
