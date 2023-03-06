package main

import (
	"os"

	"github.com/adnankobir/concourse-terraform-resource/internal/terraform"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := terraform.NewCheck(os.Stdin, os.Stderr, os.Stdout, os.Args).Execute(); err != nil {
		logrus.Errorln(err.Error())
		os.Exit(1)
	}
}
