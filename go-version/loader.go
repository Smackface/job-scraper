package main

import (
	"log"
	"os"
	"strings"
)

var (
	whitelist, blacklist []string
)

func loadConfig() ([]string, []string) {
	if len(whitelist) != 0 || len(blacklist) != 0 {
		return whitelist, blacklist
	}
	white, err := os.ReadFile("config/whitelist.txt")
	if err != nil {
		log.Fatal(err)
	}
	white_keywords := strings.Split(string(white), "\n")

	black, err := os.ReadFile("config/blacklist.txt")
	if err != nil {
		log.Fatal(err)
	}
	black_keywords := strings.Split(string(black), "\n")

	whitelist, blacklist = white_keywords, black_keywords

	return white_keywords, black_keywords
}
