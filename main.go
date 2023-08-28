package main

import (
	"github.com/chanzuckerberg/camelot/cmd"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	err := cmd.Execute()
	if err != nil && err.Error() != "" {
		logrus.Error(err)
	}
}
