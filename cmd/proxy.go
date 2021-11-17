package main

import (
	"dynproxy"
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var config *dynproxy.Config

func init() {
	configFilePath := flag.String("c", "/home/igor/code/dynproxy/cmd/config.toml", "path to configuration file.")
	flag.Parse()
	config = dynproxy.LoadConfig(*configFilePath)
	initLog(config)
}

func initLog(config *dynproxy.Config) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if &config.Global != nil {
		// TODO: parse log level
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func main() {
	log.Info().Msg("starting proxy...")
}
