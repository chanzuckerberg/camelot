package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

func remainingDays(eol string) int {
	if eol == "" { // no EOL date
		return 999
	}
	if eol == "true" { // product already EOL
		return 0
	}
	eolDate, err := time.Parse("2006-01-02", eol)
	if err != nil {
		logrus.Debugf("unable to parse date (%s): %s", eol, err.Error())
		return 999
	}
	diff := time.Until(eolDate)
	return int(diff.Hours() / 24)
}

func eolStatus(days int) types.Status {
	if days <= 30 {
		return types.StatusCritical
	}
	if days <= 90 {
		return types.StatusWarning
	}
	return types.StatusValid
}

func GetAWSProfiles() ([]string, error) {
	profiles := []string{}
	configFile := config.DefaultSharedConfigFilename()
	logrus.Debugf("Loading profiles from %s", configFile)
	f, err := ini.Load(configFile)
	if err != nil {
		return profiles, fmt.Errorf("unable to load %s", configFile)
	}

	for _, v := range f.Sections() {
		if strings.HasPrefix(v.Name(), "profile ") {
			profile, _ := strings.CutPrefix(v.Name(), "profile ")
			profiles = append(profiles, profile)
		}
	}
	return profiles, nil
}
