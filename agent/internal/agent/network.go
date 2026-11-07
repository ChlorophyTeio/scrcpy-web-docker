package agent

import (
	"fmt"
	"github.com/pion/webrtc/v4"
	"net"
	"os"
	"strconv"
	"strings"
)

// Mapped UDP ranges and explicit host candidates let the host-mode Agent run
// inside a bridge-network container without advertising unroutable container IPs.
func configureContainerICE(s *webrtc.SettingEngine, getenv func(string) string) error {
	raw := strings.TrimSpace(getenv("SCRCPYCAT_WEBRTC_HOST_IPS"))
	if raw != "" {
		ips := []string{}
		for _, part := range strings.Split(raw, ",") {
			ip := net.ParseIP(strings.TrimSpace(part))
			if ip == nil || ip.IsUnspecified() || ip.IsMulticast() {
				return fmt.Errorf("invalid SCRCPYCAT_WEBRTC_HOST_IPS")
			}
			ips = append(ips, ip.String())
		}
		s.SetNAT1To1IPs(ips, webrtc.ICECandidateTypeHost)
	}
	low, high := getenv("SCRCPYCAT_WEBRTC_UDP_MIN"), getenv("SCRCPYCAT_WEBRTC_UDP_MAX")
	if low == "" && high == "" {
		return nil
	}
	min, e := strconv.ParseUint(low, 10, 16)
	max, f := strconv.ParseUint(high, 10, 16)
	if e != nil || f != nil || min < 1024 || max < min {
		return fmt.Errorf("invalid WebRTC UDP range")
	}
	return s.SetEphemeralUDPPortRange(uint16(min), uint16(max))
}
func containerICE(s *webrtc.SettingEngine) error { return configureContainerICE(s, os.Getenv) }
