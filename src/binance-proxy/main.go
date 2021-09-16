package main

import (
	"binance-proxy/handler"
	"fmt"
	"log"
	"net/http"
	"sync"
)

func startSpotProxy(address string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.NewSpotHandler())

	log.Println("Start spot proxy !Address:", address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatal("Start spot proxy failed,err:", err)
	}
}

func startFuturesProxy(address string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.NewFuturesHandler())

	log.Println("Start futures proxy !Address:", address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatal("Start futures proxy failed,err:", err)
	}
}

// func handleSignal(wg *sync.WaitGroup) {
// 	signalChan := make(chan os.Signal)
// 	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
// 	for s := range signalChan {
// 		switch s {
// 		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
// 			wg.Done()
// 		}
// 	}
// }

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// go handleSignal(wg)

	go startSpotProxy(":8090")
	go startFuturesProxy(":8091")

	wg.Wait()

	fmt.Println("\nUser interrupted..")
}
