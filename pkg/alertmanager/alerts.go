package alertmanager

import (
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/alertmanager/types"
)

type alertResponse struct {
	Status string         `json:"status"`
	Alerts []*types.Alert `json:"data,omitempty"`
}

// ListAlerts returns a slice of Alert and an error.
func ListAlerts(logger log.Logger, alertmanagerURL string) ([]*types.Alert, error) {
	resp, err := httpRetry(logger, http.MethodGet, alertmanagerURL+"/api/v1/alerts")
	if err != nil {
		return nil, err
	}

	var alertResponse alertResponse
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if err := dec.Decode(&alertResponse); err != nil {
		return nil, err
	}

	return alertResponse.Alerts, err
}
