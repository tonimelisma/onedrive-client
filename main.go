package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/tonimelisma/onedrive-sdk-go"
)

func main() {
	// Load configuration, and initialize OAuth2 tokens
	config, err := loadConfiguration()
	config.Debug = true
	if os.IsNotExist(err) {
		fmt.Println("No configuration file found.")
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "couldn't load configuration: %v\n", err)
		os.Exit(1)
	}

	var client *http.Client

	if config.Token.AccessToken == "" {
		debug(config, "No access token found.")
		client, err = authenticate(&config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "couldn't authenticate: %v\n", err)
			os.Exit(1)
		}
	} else {
		debug(config, "Access token found.")
		client, err = validateToken(&config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't validate token: %v\n", err)
			os.Exit(1)
		}
	}

	// OAuth2 tokens ready
	fmt.Println("client:", client)

	onedrive.Hello()
}
