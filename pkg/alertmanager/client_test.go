package alertmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

const (
	jsonStatus = `
{
  "cluster": {
    "name": "01EXA9YHW49D5MR2K45MX69408",
    "peers": [
      {
        "address": "100.64.3.143:9094",
        "name": "01EXA9YHW49D5MR2K45MX69408"
      }
    ],
    "status": "ready"
  },
  "config": {
    "original": "global"
  },
  "uptime": "2021-01-30T18:47:40",
  "versionInfo": {
    "branch": "HEAD",
    "buildDate": "20200617-08:54:02",
    "buildUser": "root@dee35927357f",
    "goVersion": "go1.14.4",
    "revision": "4c6c03ebfe21009c546e4d1e9b92c371d67c021d",
    "version": "0.21.0"
  }
}
`
	jsonAlerts = `
[
  {
    "annotations": {
      "message": "This is an alert meant to ensure that the entire alerting pipeline is functional."
    },
    "endsAt": "2021-02-22T00:52:37.000Z",
    "fingerprint": "7a90bbdd1d39f61b",
    "receivers": [{"name": "healthcheck"}],
    "startsAt": "2021-01-27T16:56:37.000Z",
    "status": {
      "inhibitedBy": [],
      "silencedBy": [],
      "state": "active"
    },
    "updatedAt": "2021-02-22T00:48:37.000Z",
    "generatorURL": "https://prometheus.io/graph?g0.expr=vector%281%29\u0026g0.tab=1",
    "labels": {
      "alertname": "Watchdog",
      "prometheus": "monitoring/k8s",
      "severity": "none"
    }
  }
]`
	jsonSilences = `[
  {
    "id": "34f5f82b-b66f-456b-aff7-b556a7eafe81",
    "status": {"state": "active"},
    "updatedAt": "2021-01-11T16:10:11.000Z",
    "comment": "foo",
    "createdBy": "metalmatze",
    "endsAt": "2022-01-11T16:10:02.000Z",
    "matchers": [
      {
        "isRegex": false,
        "name": "alertname",
        "value": "KubeMemoryOvercommit"
      },
      {
        "isRegex": false,
        "name": "prometheus",
        "value": "monitoring/metalmatze"
      },
      {
        "isRegex": false,
        "name": "severity",
        "value": "warning"
      }
    ],
    "startsAt": "2021-01-11T16:10:11.000Z"
  }
]
`
)

func TestClient(t *testing.T) {
	m := http.NewServeMux()
	m.HandleFunc("/api/v2/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jsonStatus))
	})
	m.HandleFunc("/api/v2/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jsonAlerts))
	})
	m.HandleFunc("/api/v2/silences", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jsonSilences))
	})

	s := httptest.NewServer(m)
	defer s.Close()

	u, _ := url.Parse(s.URL)
	client, err := NewClient(u)
	require.NoError(t, err)

	{
		cs := "ready"
		pa := "100.64.3.143:9094"
		pn := "01EXA9YHW49D5MR2K45MX69408"
		config := "global"
		uptime := strfmt.DateTime(time.Date(2021, 01, 30, 18, 47, 40, 0, time.UTC))
		branch := "HEAD"
		buildDate := "20200617-08:54:02"
		buildUser := "root@dee35927357f"
		goVersion := "go1.14.4"
		revision := "4c6c03ebfe21009c546e4d1e9b92c371d67c021d"
		version := "0.21.0"
		expected := &models.AlertmanagerStatus{
			Cluster: &models.ClusterStatus{
				Name: "01EXA9YHW49D5MR2K45MX69408",
				Peers: []*models.PeerStatus{{
					Address: &pa,
					Name:    &pn,
				}},
				Status: &cs,
			},
			Config: &models.AlertmanagerConfig{Original: &config},
			Uptime: &uptime,
			VersionInfo: &models.VersionInfo{
				Branch:    &branch,
				BuildDate: &buildDate,
				BuildUser: &buildUser,
				GoVersion: &goVersion,
				Revision:  &revision,
				Version:   &version,
			},
		}

		status, err := client.Status(context.Background())
		require.NoError(t, err)
		require.Equal(t, expected, status)
	}
	{
		expected := []*types.Alert{{
			Alert: model.Alert{
				Labels: model.LabelSet{
					model.LabelName("alertname"):  model.LabelValue("Watchdog"),
					model.LabelName("prometheus"): model.LabelValue("monitoring/k8s"),
					model.LabelName("severity"):   model.LabelValue("none"),
				},
				Annotations: model.LabelSet{
					model.LabelName("message"): model.LabelValue("This is an alert meant to ensure that the entire alerting pipeline is functional."),
				},
				StartsAt:     time.Date(2021, 01, 27, 16, 56, 37, 0, time.UTC),
				EndsAt:       time.Date(2021, 02, 22, 0, 52, 37, 0, time.UTC),
				GeneratorURL: "https://prometheus.io/graph?g0.expr=vector%281%29&g0.tab=1",
			},
			UpdatedAt: time.Date(2021, 02, 22, 0, 48, 37, 0, time.UTC),
			Timeout:   false,
		}}

		alerts, err := client.ListAlerts(context.Background(), "", false)
		require.NoError(t, err)
		require.Equal(t, expected, alerts)
	}
	{
		expected := []*types.Silence{{
			ID:        "34f5f82b-b66f-456b-aff7-b556a7eafe81",
			CreatedBy: "metalmatze",
			Comment:   "foo",
			StartsAt:  time.Date(2021, 01, 11, 16, 10, 11, 0, time.UTC),
			EndsAt:    time.Date(2022, 01, 11, 16, 10, 02, 0, time.UTC),
			UpdatedAt: time.Date(2021, 01, 11, 16, 10, 11, 0, time.UTC),
			Matchers:  types.Matchers{},
			Status: types.SilenceStatus{
				State: types.SilenceStateActive,
			},
		}}

		alerts, err := client.ListSilences(context.Background())
		require.NoError(t, err)
		require.Equal(t, expected, alerts)
	}
}
