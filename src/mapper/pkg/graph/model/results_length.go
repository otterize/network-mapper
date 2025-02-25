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

type AWSOperationResults []AWSOperation

func (c AWSOperationResults) Length() int {
	return len(c)
}

type AzureOperationResults []AzureOperation

func (c AzureOperationResults) Length() int {
	return len(c)
}

func (c CaptureTCPResults) Length() int {
	return len(c.Results)
}

func (c TrafficLevelResults) Length() int {
	return len(c.Results)
}
