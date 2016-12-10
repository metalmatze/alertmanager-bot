package main

import (
	"sort"
	"testing"
	"time"

	"github.com/prometheus/alertmanager/types"
	"github.com/stretchr/testify/assert"
)

func TestByStatus(t *testing.T) {
	s1 := types.Silence{EndsAt: time.Now().Add(-2 * time.Minute)}
	s2 := types.Silence{EndsAt: time.Now().Add(-1 * time.Minute)}
	s3 := types.Silence{EndsAt: time.Now().Add(1 * time.Minute)}
	s4 := types.Silence{EndsAt: time.Now().Add(2 * time.Minute)}

	bs := ByStatus{s1}
	sort.Sort(bs)
	assert.Equal(t, s1, bs[0])

	bs = ByStatus{s4, s1}
	sort.Sort(bs)
	assert.Equal(t, s4, bs[0])
	assert.Equal(t, s1, bs[1])

	bs = ByStatus{s3, s2}
	sort.Sort(bs)
	assert.Equal(t, s3, bs[0])
	assert.Equal(t, s2, bs[1])

	bs = ByStatus{s4, s2, s3, s1}
	sort.Sort(bs)
	assert.Equal(t, s4, bs[0])
	assert.Equal(t, s3, bs[1])
	assert.Equal(t, s2, bs[2])
	assert.Equal(t, s1, bs[3])
}

func TestResolved(t *testing.T) {
	s := types.Silence{}
	assert.False(t, Resolved(s))

	s.EndsAt = time.Now().Add(time.Minute)
	assert.False(t, Resolved(s))

	s.EndsAt = time.Now().Add(-1 * time.Minute)
	assert.True(t, Resolved(s))
}
