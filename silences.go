package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prometheus/alertmanager/types"
)

type silencesResponse struct {
	Data   []types.Silence `json:"data"`
	Status string          `json:"status"`
}

func listSilences(c Config) ([]types.Silence, error) {
	url := c.AlertmanagerURL + "/api/v1/silences"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	var silencesResponse silencesResponse
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if err := dec.Decode(&silencesResponse); err != nil {
		return nil, err
	}

	return silencesResponse.Data, err
}

// SilenceMessage converts a silences to a message string
func SilenceMessage(s types.Silence) string {
	var alertname, matchers string

	for _, m := range s.Matchers {
		if m.Name == "alertname" {
			alertname = m.Value
		} else {
			matchers = matchers + fmt.Sprintf(` %s="%s"`, m.Name, m.Value)
		}
	}

	return fmt.Sprintf(
		"%s 🔕\n```%s```\n",
		alertname,
		strings.TrimSpace(matchers),
	)
}
