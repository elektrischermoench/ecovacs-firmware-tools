package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/denysvitali/ecovacs-firmware-tools/cmd"
)

func main() {
	log.SetLevel(log.InfoLevel)
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
