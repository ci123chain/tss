package logic

import (
	"log"
	"os"
	"os/signal"
)

func CatchCmdSignals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	for sig := range signals {
		// sig is a ^C, handle it
		switch sig {
		case os.Interrupt:
			log.Println("Bye!")
			os.Exit(0)
		}
	}
}
