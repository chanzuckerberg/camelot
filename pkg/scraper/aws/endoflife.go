package aws

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/pkg/errors"
)

func endOfLife(entity string) (*[]types.ProductCycle, error) {
	res, err := http.Get(fmt.Sprintf("https://endoflife.date/api/%s.json", entity))
	if err != nil {
		return nil, fmt.Errorf("failed to get end of life data")
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("error getting end of life data: %s", res.Status)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body")
	}
	productCycles := []types.ProductCycle{}
	err = json.Unmarshal(body, &productCycles)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal end of life data")
	}
	return &productCycles, nil
}
