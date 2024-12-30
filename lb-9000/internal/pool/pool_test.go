package pool

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetIpFromUrl(t *testing.T) {
	raw := "http://10-244-0-6.server-service.default.svc.cluster.local:8080/"

	ip, err := getIpFromHost(raw)
	assert.Nil(t, err)
	assert.Equal(t, "10.244.0.6", ip)

	raw = "10-244-0-6.server-service.default.svc.cluster.local:8080/"

	ip, err = getIpFromHost(raw)
	assert.Nil(t, err)

	assert.Equal(t, "10.244.0.6", ip)
}
