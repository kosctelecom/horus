package agent

import "horus-core/log"

// PingCollector is a prometheus collector
type PingCollector struct {
	*PromCollector
}

// Push converts a ping measure to prometheus samples and pushes them to the sample queue.
func (c *PingCollector) Push(meas PingMeasure) {
	log.Debug2f(">> posting ping measures for %s at %v", meas.IpAddr, meas.Stamp)
	ping_min := PromSample{
		Name:  "ping_min_duration_seconds",
		Desc:  "min ping RTT time on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"host":          meas.Hostname,
			"ip_address":    meas.IpAddr,
			"device_type":   meas.Category,
			"device_vendor": meas.Vendor,
			"device_model":  meas.Model,
		},
		Value: meas.Min,
	}
	c.promSamples <- &ping_min

	ping_max := PromSample{
		Name:  "ping_max_duration_seconds",
		Desc:  "max ping RTT time on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"host":          meas.Hostname,
			"ip_address":    meas.IpAddr,
			"device_type":   meas.Category,
			"device_vendor": meas.Vendor,
			"device_model":  meas.Model,
		},
		Value: meas.Max,
	}
	c.promSamples <- &ping_max

	ping_avg := PromSample{
		Name:  "ping_avg_duration_seconds",
		Desc:  "average ping RTT time on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"host":          meas.Hostname,
			"ip_address":    meas.IpAddr,
			"device_type":   meas.Category,
			"device_vendor": meas.Vendor,
			"device_model":  meas.Model,
		},
		Value: meas.Avg,
	}
	c.promSamples <- &ping_avg

	ping_loss := PromSample{
		Name:  "ping_loss_ratio",
		Desc:  "ping packet loss ratio on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"host":          meas.Hostname,
			"ip_address":    meas.IpAddr,
			"device_type":   meas.Category,
			"device_vendor": meas.Vendor,
			"device_model":  meas.Model,
		},
		Value: meas.Loss,
	}
	c.promSamples <- &ping_loss
}
