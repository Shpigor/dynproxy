package main

import (
	"context"
	"dynproxy"
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
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
	sigOsChan := make(chan int)
	go handleSysSignals(sigOsChan)
	mainCtx, mainCancelFn := context.WithCancel(context.Background())
	manager := dynproxy.NewContextManager(mainCtx)
	dynproxy.InitBalancers(mainCtx, config)
	manager.InitFrontends(config)
	<-sigOsChan
	mainCancelFn()
	log.Info().Msg("proxy stopped")
}

func handleSysSignals(exitChan chan int) {
	sysSignalChanel := make(chan os.Signal, 1)
	signal.Notify(sysSignalChanel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for {
		s := <-sysSignalChanel
		switch s {
		case syscall.SIGHUP: // kill -SIGHUP PID
			log.Info().Msg("Signal hang up triggered.")
		case syscall.SIGINT: // kill -SIGINT PID or Ctrl+c
			log.Info().Msg("Signal interrupt triggered.")
			exitChan <- 0
		case syscall.SIGTERM: // kill -SIGTERM PID
			log.Info().Msg("Signal terminte triggered.")
			exitChan <- 0
		case syscall.SIGQUIT: // kill -SIGQUIT PID
			log.Info().Msg("Signal quit triggered.")
			exitChan <- 0
		default:
			log.Info().Msg("Unknown signal.")
			exitChan <- 1
		}
	}
}
