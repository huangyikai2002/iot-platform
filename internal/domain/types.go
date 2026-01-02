package domain

type DeviceMessage struct {
	DeviceID    string  `json:"device_id"`
	Token       string  `json:"token"`
	Temperature float64 `json:"temperature"`
	Pressure    float64 `json:"pressure"`
	Timestamp   int64   `json:"timestamp"`
}
