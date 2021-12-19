package main

import (
	"distributed-rental/projects/auth/internal"
	"flag"
	"github.com/dgraph-io/badger/v3"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	addrF := flag.String("addr", "localhost:3000", "addr to listen on")
	jwtSecretPath := flag.String("jwt-secret-path", "/etc/jwt-secret", "path to jwt secret")

	flag.Parse()

	jwtSecret, err := ioutil.ReadFile(*jwtSecretPath)
	if err != nil {
		log.Fatal(err)
	}

	db, err := badger.Open(badger.DefaultOptions("/var/auth_db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	userIDSequence, err := db.GetSequence([]byte("user_id_sequence"), 100_000)
	if err != nil {
		log.Fatal(err)
	}

	userService := internal.NewUserService(db, userIDSequence)

	httpServer := internal.NewHttpServer(*addrF, userService, jwtSecret, logger.Sugar())

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
