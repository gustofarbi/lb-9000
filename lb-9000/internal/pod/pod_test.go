package pod

import (
	"slices"
	"testing"
)

func TestDeleteSlice(t *testing.T) {
	str := []string{"foo", "bar", "baz"}

	slices.Delete(str, 1, 2)

	t.Log(str)
}
