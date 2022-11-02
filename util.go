package main

import "log"

func debug(config Configuration, message string) {
	if config.Debug {
		log.Println(message)
	}
}
