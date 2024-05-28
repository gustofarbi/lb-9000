package pool

import (
	"testing"
)

func TestGetIpFromUrl(t *testing.T) {
	raw := "http://10-244-0-6.server-service.default.svc.cluster.local:8080/"

	ip, err := getIpFromHost(raw)
	if err != nil {
		t.Fatal(err)
	}

	if ip != "10.244.0.6" {
		t.Errorf("expected ip to be 10-244-0-6, got %s", ip)
	}

	raw = "10-244-0-6.server-service.default.svc.cluster.local:8080/"

	ip, err = getIpFromHost(raw)
	if err != nil {
		t.Fatal(err)
	}

	if ip != "10.244.0.6" {
		t.Errorf("expected ip to be 10-244-0-6, got %s", ip)
	}
}
