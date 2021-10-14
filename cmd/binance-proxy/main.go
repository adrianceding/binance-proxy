package main

import (
	"binance-proxy/internal/handler"
	"binance-proxy/internal/service"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "net/http/pprof"

	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

func startProxy(ctx context.Context, port int, class service.Class, disablefakekline bool, alwaysshowforwards bool) {
	mux := http.NewServeMux()
	address := fmt.Sprintf(":%d", port)
	mux.HandleFunc("/", handler.NewHandler(ctx, class, !disablefakekline, alwaysshowforwards))

	log.Infof("%s websocket proxy starting on port %d.", class, port)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatalf("%s websocket proxy start failed (error: %s).", class, err)
	}
}

func handleSignal() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for s := range signalChan {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			cancel()
		}
	}
}

type Config struct {
	Verbose            []bool `short:"v" long:"verbose" env:"BPX_VERBOSE" description:"Verbose output (increase with -vv)"`
	SpotAddress        int    `short:"p" long:"port-spot" env:"BPX_PORT_SPOT" description:"Port to which to bind for SPOT markets" default:"8090"`
	FuturesAddress     int    `short:"t" long:"port-futures" env:"BPX_PORT_FUTURES" description:"Port to which to bind for FUTURES markets" default:"8091"`
	DisableFakeKline   bool   `short:"c" long:"disable-fake-candles" env:"BPX_DISABLE_FAKE_CANDLES" description:"Disable generation of fake candles (ohlcv) when sockets have not delivered data yet"`
	DisableSpot        bool   `short:"s" long:"disable-spot" env:"BPX_DISABLE_SPOT" description:"Disable proxying spot markets"`
	DisableFutures     bool   `short:"f" long:"disable-futures" env:"BPX_DISABLE_FUTURES" description:"Disable proxying futures markets"`
	AlwaysShowForwards bool   `short:"a" long:"always-show-forwards" env:"BPX_ALWAYS_SHOW_FORWARDS" description:"Always show requests forwarded via REST even if verbose is disabled"`
}

var (
	config      Config
	parser             = flags.NewParser(&config, flags.Default)
	Version     string = "develop"
	Buildtime   string = "undefined"
	ctx, cancel        = context.WithCancel(context.Background())
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	log.Infof("Binance proxy version %s, build time %s", Version, Buildtime)

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			log.Fatalf("%s - %s", err, flagsErr.Type)
		}
	}

	if len(config.Verbose) >= 2 {
		log.SetLevel(log.TraceLevel)
	} else if len(config.Verbose) == 1 {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if log.GetLevel() > log.InfoLevel {
		log.Infof("Set level to %s", log.GetLevel())
	}

	if config.DisableSpot && config.DisableFutures {
		log.Fatal("can't start if both SPOT and FUTURES are disabled!")
	}

	if !config.DisableFakeKline {
		log.Infof("Fake candles are enabled for faster processing, the feature can be disabled with --disable-fake-candles or -c")
	}

	if config.AlwaysShowForwards {
		log.Infof("Always show forwards is enabled, all API requests, that can't be served from websockets cached will be logged.")
	}

	go handleSignal()

	if !config.DisableSpot {
		go startProxy(ctx, config.SpotAddress, service.SPOT, config.DisableFakeKline, config.AlwaysShowForwards)
	}
	if !config.DisableFutures {
		go startProxy(ctx, config.FuturesAddress, service.FUTURES, config.DisableFakeKline, config.AlwaysShowForwards)
	}
	<-ctx.Done()

	log.Info("SIGINT received, aborting ...")
}
