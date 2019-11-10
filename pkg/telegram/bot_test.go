package telegram

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

func Test_internalAlerts(t *testing.T) {
	started := strfmt.DateTime(time.Now().Add(-5 * time.Minute))
	updated := strfmt.DateTime(time.Now().Add(-1 * time.Minute))
	ends := strfmt.DateTime(time.Now().Add(3 * time.Minute))

	tests := []struct {
		Name string
		In   models.GettableAlerts
		Out  []*types.Alert
	}{{
		Name: "simple",
		In: models.GettableAlerts{{
			Receivers: nil,
			Status:    nil,
			Alert: models.Alert{
				GeneratorURL: "This is the generator URL",
				Labels: models.LabelSet{
					"alertname": "Watchdog",
					"namespace": "monitoring",
				},
			},
			StartsAt:  &started,
			UpdatedAt: &updated,
			EndsAt:    &ends,
		}},
		Out: []*types.Alert{{
			Alert: model.Alert{
				Labels: model.LabelSet{
					"alertname": "Watchdog",
					"namespace": "monitoring",
				},
				Annotations:  nil,
				StartsAt:     time.Time(started),
				EndsAt:       time.Time(ends),
				GeneratorURL: "This is the generator URL",
			},
			UpdatedAt: time.Time(updated),
			Timeout:   false,
		}},
	}}
	for _, tc := range tests {
		alerts := internalAlerts(tc.In)
		assert.Len(t, alerts, len(tc.Out))

		for i, out := range tc.Out {
			assert.Equal(t, out, alerts[i])
		}
	}
}
