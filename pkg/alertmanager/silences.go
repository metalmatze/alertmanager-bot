package alertmanager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/hako/durafmt"
	"github.com/prometheus/alertmanager/types"
)

type silencesResponse struct {
	Data   []types.Silence `json:"data"`
	Status string          `json:"status"`
}

// ListSilences returns a slice of Silence and an error.
func ListSilences(logger log.Logger, alertmanagerURL string) ([]types.Silence, error) {
	resp, err := httpRetry(logger, http.MethodGet, alertmanagerURL+"/api/v1/silences")
	if err != nil {
		return nil, err
	}

	var silencesResponse silencesResponse
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if err := dec.Decode(&silencesResponse); err != nil {
		return nil, err
	}

	silences := silencesResponse.Data
	sort.Slice(silences, func(i, j int) bool {
		return silences[i].EndsAt.After(silences[j].EndsAt)
	})

	return silences, err
}

// SilenceMessage converts a silences to a message string
func SilenceMessage(s types.Silence) string {
	var alertname, emoji, matchers, duration string

	for _, m := range s.Matchers {
		if m.Name == "alertname" {
			alertname = m.Value
		} else {
			matchers = matchers + fmt.Sprintf(` %s="%s"`, m.Name, m.Value)
		}
	}

	resolved := Resolved(s)
	if !resolved {
		emoji = " 🔕"
		duration = fmt.Sprintf(
			"*Started*: %s ago\n*Ends:* %s\n",
			durafmt.Parse(time.Since(s.StartsAt)),
			durafmt.Parse(time.Since(s.EndsAt)),
		)
	} else {
		duration = fmt.Sprintf(
			"*Ended*: %s ago\n*Duration*: %s",
			durafmt.Parse(time.Since(s.EndsAt)),
			durafmt.Parse(s.EndsAt.Sub(s.StartsAt)),
		)
	}

	return fmt.Sprintf(
		"%s%s\n```%s```\n%s\n",
		alertname, emoji,
		strings.TrimSpace(matchers),
		duration,
	)
}

// Resolved returns if a silence is resolved by EndsAt
func Resolved(s types.Silence) bool {
	if s.EndsAt.IsZero() {
		return false
	}
	return !s.EndsAt.After(time.Now())
}
