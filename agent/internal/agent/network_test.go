package agent

import (
	"github.com/pion/webrtc/v4"
	"testing"
)

func TestContainerICEConfiguration(t *testing.T) {
	for _, tc := range []struct {
		ips, low, high string
		bad            bool
	}{{"", "", "", false}, {"192.0.2.10", "40000", "40100", false}, {"192.0.2.10,2001:db8::10", "40000", "40100", false}, {"example.com", "", "", true}, {"0.0.0.0", "", "", true}, {"", "40000", "", true}, {"", "40100", "40000", true}, {"", "0", "1", true}} {
		s := webrtc.SettingEngine{}
		e := configureContainerICE(&s, func(k string) string {
			switch k {
			case "SCRCPYCAT_WEBRTC_HOST_IPS":
				return tc.ips
			case "SCRCPYCAT_WEBRTC_UDP_MIN":
				return tc.low
			default:
				return tc.high
			}
		})
		if (e != nil) != tc.bad {
			t.Fatalf("%+v: %v", tc, e)
		}
	}
}
