package config

import (
	"flag"
)

var (
	FlagRunAddr string
	FlagBaseURL string
)

func ParseFlags() {
	flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&FlagBaseURL, "b", "http://localhost:8080", "base URL for shortened links")
	flag.Parse()
}
