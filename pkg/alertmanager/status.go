package alertmanager

import (
	"context"

	"github.com/prometheus/alertmanager/api/v2/client/general"
	"github.com/prometheus/alertmanager/api/v2/models"
)

func (c Client) Status(ctx context.Context) (*models.AlertmanagerStatus, error) {
	status, err := c.alertmanager.General.GetStatus(general.NewGetStatusParams().WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return status.Payload, nil
}
