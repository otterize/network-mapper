package config

import "time"

/*
Shared config keys between all reporter components - DNS Sniffer, Kafka watcher, Istio watcher
*/

const (
	MapperApiUrlKey         = "mapper-api-url"
	MapperApiUrlDefault     = "http://mapper:9090/query"
	DebugKey                = "debug"
	DebugDefault            = false
	ReportIntervalKey       = "report-interval"
	ReportIntervalDefault   = 10 * time.Second
	CallsTimeoutKey         = "calls-timeout"
	CallsTimeoutDefault     = 5 * time.Second
	CooldownIntervalKey     = "cooldown-interval"
	CooldownIntervalDefault = 10 * time.Second
	EnvPrefix               = "OTTERIZE"
)
