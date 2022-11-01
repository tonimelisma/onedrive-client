package main

import (
	"fmt"
	"os"

	"github.com/tonimelisma/onedrive-sdk-go"
)

func main() {
	onedrive.Hello()

	config, err := loadConfiguration()
	if !os.IsNotExist(err) && err != nil {
		fmt.Fprintf(os.Stderr, "couldn't load configuration: %v\n", err)
		os.Exit(1)
	}

	if config.AccessToken == "" {
		_, err = authenticate(&config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "couldn't authenticate: %v\n", err)
			os.Exit(1)
		}
	} else {
		validateToken(&config)
	}
}
