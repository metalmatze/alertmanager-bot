package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type statusResponse struct {
	Status string `json:"status"`
	Data   struct {
		Uptime      time.Time `json:"uptime"`
		VersionInfo struct {
			Branch    string `json:"branch"`
			BuildDate string `json:"buildDate"`
			BuildUser string `json:"buildUser"`
			GoVersion string `json:"goVersion"`
			Revision  string `json:"revision"`
			Version   string `json:"version"`
		} `json:"versionInfo"`
	} `json:"data"`
}

func status(alertmanagerURL string) (statusResponse, error) {
	var statusResponse statusResponse

	resp, err := http.Get(alertmanagerURL + "/api/v1/status")
	if err != nil {
		return statusResponse, err
	}

	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if err := dec.Decode(&statusResponse); err != nil {
		return statusResponse, err
	}

	return statusResponse, nil
}
