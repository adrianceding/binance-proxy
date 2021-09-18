package main

import (
	"binance-proxy/handler"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func startSpotProxy(address string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.NewSpotHandler())

	log.Info("Start spot proxy !Address:", address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatal("Start spot proxy failed,err:", err)
	}
}

func startFuturesProxy(address string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.NewFuturesHandler())

	log.Info("Start futures proxy !Address:", address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatal("Start futures proxy failed,err:", err)
	}
}

func handleSignal(wg *sync.WaitGroup) {
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for s := range signalChan {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			wg.Done()
		}
	}
}

var flagSpotAddress string
var flagFuturesAddress string
var flagDebug bool

func main() {
	flag.StringVar(&flagSpotAddress, "s", ":8090", "spot bind address.")
	flag.StringVar(&flagFuturesAddress, "f", ":8091", "futures bind address.")
	flag.BoolVar(&flagDebug, "v", false, "print debug log.")
	flag.Parse()

	if flagDebug {
		log.SetLevel(log.DebugLevel)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go handleSignal(wg)

	go startSpotProxy(flagSpotAddress)
	go startFuturesProxy(flagFuturesAddress)

	wg.Wait()

	log.Info("\nUser interrupted..")
}
