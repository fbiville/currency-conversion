package main

import (
	"fmt"
	"github.com/fbiville/currency-conversion/pkg/currency"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		panic("missing APILayer API key")
	}
	converter := currency.NewConverter(
		"https://api.apilayer.com",
		apiKey)
	controller := currency.NewController(converter)

	server := &http.Server{
		Addr:    ":0",
		Handler: controller,
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic("could not start listener")
	}
	fmt.Printf("http://%s\n", listener.Addr().String())

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		server.Close()
	}()
	server.Serve(listener)
}
