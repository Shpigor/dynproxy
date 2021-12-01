package dynproxy

type StatsManager struct {
	MemoryUsage int32
	CPUUsage    float32
	ConfigPath  string
	ConfigHash  string
	activeStats ActiveStats
}

type ActiveStats struct {
	FrontendsStats map[string]FrontendStats
	BalancersStats map[string]BalancerStats
}

type FrontendStats struct {
	Name               string
	ActiveSessions     uint64
	MaxActiveSessions  uint64
	MaxSessions        uint64
	TotalSentBytes     uint64
	TotalReceivedBytes uint64
	Bandwidth          float64
}

type BalancerStats struct {
	Name string
}
