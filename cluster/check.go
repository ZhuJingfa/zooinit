package cluster

import (
	"encoding/json"
	"os"
	"time"
)

// Health check
type ServiceCheck interface {
	IsHealth() bool

	Members()

	AddMember() error

	DelMember() error
}

type ClusterMember struct {
	Name     string `json:"name"`
	Update   string `json:"update"`
	Localip  string `json:"localip"`
	Hostname string `json:"hostname"`
	State    bool   `json:"state"`
	Failed   int    `json:"failed"`
}

func NewClusterMember(Name, Localip string, State bool, Failed int) *ClusterMember {
	return &ClusterMember{Name: Name, Localip: Localip, State: State, Failed: Failed, Update: time.Now().Format(time.RFC3339), Hostname: os.Getenv("HOSTNAME")}
}

func (m *ClusterMember) ToJson() string {
	b, err := json.Marshal(m)
	if err == nil {
		return string(b)
	} else {
		return ""
	}
}

func (m *ClusterMember) IsHealth() bool {
	return m.State
}
