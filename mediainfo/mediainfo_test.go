package mediainfo

import (
	"testing"
)

func TestMediainfo(t *testing.T) {
	mi, ok := MediaMap["adsist.ai"]
	if !ok {
		t.Error("adsist.ai should be ok.")
	}
	t.Logf("mi: %v\n", mi)
}
