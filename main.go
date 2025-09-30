package main

import (
	"os"

	"github.com/denysvitali/ecovacs-firmware-tools/cmd"
	log "github.com/sirupsen/logrus"
)

func main() {
	// Configure logrus
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Execute root command
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}