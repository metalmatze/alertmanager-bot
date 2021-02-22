package alertmanager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/prometheus/alertmanager/api/v2/client/silence"
	"github.com/prometheus/alertmanager/types"
)

func (c *Client) ListSilences(ctx context.Context) ([]*types.Silence, error) {
	getSilences, err := c.alertmanager.Silence.GetSilences(silence.NewGetSilencesParams().WithContext(ctx))
	if err != nil {
		return nil, err
	}

	silences := make([]*types.Silence, 0, len(getSilences.Payload))
	for _, s := range getSilences.Payload {
		var matchers = make([]*types.Matcher, 0, len(s.Matchers))
		for _, m := range matchers {
			matchers = append(matchers, &types.Matcher{
				Name:    m.Name,
				Value:   m.Value,
				IsRegex: m.IsRegex,
			})
		}

		silences = append(silences, &types.Silence{
			ID:        *s.ID,
			StartsAt:  time.Time(*s.StartsAt),
			EndsAt:    time.Time(*s.EndsAt),
			UpdatedAt: time.Time(*s.UpdatedAt),
			CreatedBy: *s.CreatedBy,
			Comment:   *s.Comment,
			Matchers:  matchers,
			Status: types.SilenceStatus{
				State: types.SilenceState(*s.Status.State),
			},
		})
	}

	return silences, nil
}

// SilenceMessage converts a silences to a message string.
func SilenceMessage(s *types.Silence) string {
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

// Resolved returns if a silence is resolved by EndsAt.
func Resolved(s *types.Silence) bool {
	if s.EndsAt.IsZero() {
		return false
	}
	return !s.EndsAt.After(time.Now())
}
