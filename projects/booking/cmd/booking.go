package main

import (
	"distributed-rental/projects/booking/internal"
	"flag"
	"github.com/dgraph-io/badger/v3"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	addrF := flag.String("addr", "localhost:3002", "addr to listen on")

	flag.Parse()

	db, err := badger.Open(badger.DefaultOptions("~/var/booking_db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	bookingIDSequence, err := db.GetSequence([]byte("booking_id_sequence"), 100_000)
	if err != nil {
		log.Fatal(err)
	}

	bookingService := &internal.BookingService{
		DB: db,
		BookingIDSequence: bookingIDSequence,
		Logger: logger,
	}

	httpServer := internal.NewHttpServer(*addrF, bookingService, logger.Sugar())

	go func() {
		err := httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Sugar().Errorf("error closing server: %v", err)
		}
	}()

	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGSTOP)

	<-signals
	httpServer.Close()
}
