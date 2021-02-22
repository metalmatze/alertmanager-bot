package alertmanager

import (
	"net/url"
	"path"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
)

type Client struct {
	alertmanager *client.Alertmanager
}

func NewClient(url *url.URL) (*Client, error) {
	alertmanagerPath := url.Path
	if !strings.HasSuffix(alertmanagerPath, "/api/v2") {
		alertmanagerPath = path.Join(alertmanagerPath, "/api/v2")
	}

	return &Client{
		alertmanager: client.NewHTTPClientWithConfig(strfmt.Default,
			client.DefaultTransportConfig().
				WithSchemes([]string{url.Scheme}).
				WithHost(url.Host).
				WithBasePath(alertmanagerPath),
		),
	}, nil
}
