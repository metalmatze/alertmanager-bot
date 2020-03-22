package alertmanager

import (
	"net/url"
	"time"

	"github.com/go-openapi/strfmt"
	alertmanagerclient "github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
)

var (
	yes = true
	no  = false
)

// Alertmanager is mostly a wrapper around the swagger generated v2 client,
// doing some conversion of types, adding additional metrics
// and setting some defaults for the bot.
type Alertmanager struct {
	client *alertmanagerclient.Alertmanager
}

// New create a new Alertmanager wrapper for the bot.
func New(u *url.URL) *Alertmanager {
	cfg := alertmanagerclient.DefaultTransportConfig().WithSchemes([]string{u.Scheme}).WithHost(u.Host).WithBasePath(u.Path)
	client := alertmanagerclient.NewHTTPClientWithConfig(strfmt.NewFormats(), cfg)

	return &Alertmanager{client}
}

// ListAlerts by making a request to Alertmanager with some sane defaults for the bot.
func (a *Alertmanager) ListAlerts() ([]*types.Alert, error) {
	resp, err := a.client.Alert.GetAlerts(alert.NewGetAlertsParams().WithActive(&yes).WithSilenced(&no))
	if err != nil {
		return nil, err
	}

	var alerts []*types.Alert
	for _, a := range resp.GetPayload() {
		labels := map[model.LabelName]model.LabelValue{}
		for n, v := range a.Labels {
			labels[model.LabelName(n)] = model.LabelValue(v)
		}
		annotations := map[model.LabelName]model.LabelValue{}
		for n, v := range a.Annotations {
			labels[model.LabelName(n)] = model.LabelValue(v)
		}

		alerts = append(alerts, &types.Alert{
			Alert: model.Alert{
				Labels:       labels,
				Annotations:  annotations,
				StartsAt:     time.Time(*a.StartsAt),
				EndsAt:       time.Time(*a.EndsAt),
				GeneratorURL: string(a.GeneratorURL),
			},
			UpdatedAt: time.Time(*a.UpdatedAt),
			//Timeout:   false,
		})
	}

	return alerts, nil
}
