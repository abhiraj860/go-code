package inventory

type AlertConfig struct {
	threshold int
	listener AlertListener
}

func NewAlertConfig(threshold int, listener AlertListener) *AlertConfig {
	return &AlertConfig{
		threshold: threshold,
		listener: listener,
	}
}

func (c *AlertConfig) GetThreshold() int {
	return c.threshold
}

func (c *AlertConfig) GetListener() AlertListener {
	return c.listener
}