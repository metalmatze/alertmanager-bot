package alertmanager

import (
	"testing"
	"time"

	"github.com/prometheus/alertmanager/types"
	"github.com/stretchr/testify/assert"
)

func TestResolved(t *testing.T) {
	s := types.Silence{}
	assert.False(t, Resolved(s))

	s.EndsAt = time.Now().Add(time.Minute)
	assert.False(t, Resolved(s))

	s.EndsAt = time.Now().Add(-1 * time.Minute)
	assert.True(t, Resolved(s))
}
