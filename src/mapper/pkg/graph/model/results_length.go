package model

func (c CaptureResults) Length() int {
	return len(c.Results)
}

func (c KafkaMapperResults) Length() int {
	return len(c.Results)
}

func (c SocketScanResults) Length() int {
	return len(c.Results)
}

func (c IstioConnectionResults) Length() int {
	return len(c.Results)
}
