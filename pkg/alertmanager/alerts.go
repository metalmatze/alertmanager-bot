package alertmanager

import (
	"context"
	"time"

	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
)

func (c *Client) ListAlerts(ctx context.Context, receiver string, silenced bool) ([]*types.Alert, error) {
	getAlerts, err := c.alertmanager.Alert.GetAlerts(alert.NewGetAlertsParams().WithContext(ctx).
		WithReceiver(&receiver).
		WithSilenced(&silenced),
	)
	if err != nil {
		return nil, err
	}

	alerts := make([]*types.Alert, 0, len(getAlerts.Payload))
	for _, a := range getAlerts.Payload {
		labels := make(model.LabelSet, len(a.Labels))
		for name, value := range a.Labels {
			labels[model.LabelName(name)] = model.LabelValue(value)
		}
		annotations := make(model.LabelSet, len(a.Annotations))
		for name, value := range a.Annotations {
			annotations[model.LabelName(name)] = model.LabelValue(value)
		}

		endsAt := time.Time{}
		if a.EndsAt != nil {
			endsAt = time.Time(*a.EndsAt)
		}
		updatedAt := time.Time{}
		if a.UpdatedAt != nil {
			updatedAt = time.Time(*a.UpdatedAt)
		}

		alerts = append(alerts, &types.Alert{
			Alert: model.Alert{
				Labels:       labels,
				Annotations:  annotations,
				StartsAt:     time.Time(*a.StartsAt),
				EndsAt:       endsAt,
				GeneratorURL: a.GeneratorURL.String(),
			},
			UpdatedAt: updatedAt,
			Timeout:   false,
		})
	}

	return alerts, nil
}
