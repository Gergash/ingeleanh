package sim

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadLaurelesCSV(t *testing.T) {
	path := "../../data/LAURELES V2 AB 21.csv"
	if _, err := os.Stat(path); err != nil {
		t.Skip("laureles csv not in workspace")
	}
	feed, err := LoadLaurelesCSV(path, 5)
	require.NoError(t, err)
	ev := feed.NextEvent()
	require.Equal(t, "iot_event", ev["type"])
	require.Equal(t, "sensor-access-gate", ev["device_id"])
	require.NotEmpty(t, ev["payload_summary"])
}
