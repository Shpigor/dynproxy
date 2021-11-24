package main

import (
	"context"
	"dynproxy"
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"path/filepath"
	"sync"
)

var config dynproxy.Config

func init() {
	configFilePath := flag.String("c", "./cmd/config.toml", "path to configuration file.")
	flag.Parse()
	absConfigPath, err := filepath.Abs(*configFilePath)
	if err != nil {
		log.Fatal().Msgf("got error while loading config file:%+v", err)
	}
	config = dynproxy.LoadConfig(absConfigPath)
	initLog(config)
}

func initLog(config dynproxy.Config) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if &config.Global != nil {
		// TODO: parse log level
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func main() {
	log.Info().Msg("starting proxy...")
	group := sync.WaitGroup{}
	group.Add(1)
	mainCtx, _ := context.WithCancel(context.Background())
	manager := dynproxy.NewContextManager(mainCtx)
	dynproxy.InitBalancers(mainCtx, config)
	manager.InitFrontends(config)
	group.Wait()
}
