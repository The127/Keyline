package quorum

import (
	"Keyline/internal/config"
	"context"
	"fmt"
)

type LeaderChangeCallback func(isLeader bool)

type LeaderElectionFactory struct {
	callback LeaderChangeCallback
}

func NewLeaderElectionFactory() *LeaderElectionFactory {
	return &LeaderElectionFactory{}
}

func (f *LeaderElectionFactory) OnLeaderChange(callback LeaderChangeCallback) *LeaderElectionFactory {
	f.callback = callback
	return f
}

func (f *LeaderElectionFactory) Build(leaderElectionConfig config.LeaderElectionConfig) LeaderElection {
	switch leaderElectionConfig.Mode {
	case config.LeaderElectionModeNone:
		return NewNoLeaderElection(f.callback)

	case config.LeaderElectionModeRaft:
		return NewRaftLeaderElection(f.callback)

	default:
		panic(fmt.Sprintf("leader election mode %s not supported", leaderElectionConfig.Mode))
	}
}

type LeaderElection interface {
	Start(ctx context.Context) error
	Stop() error
	IsLeader() bool
}
