package quorum

import "context"

type noLeaderElection struct {
	callback LeaderChangeCallback
}

func NewNoLeaderElection(callback LeaderChangeCallback) LeaderElection {
	return &noLeaderElection{
		callback: callback,
	}
}

func (n *noLeaderElection) Start(_ context.Context) error {
	n.callback(true)
	return nil
}

func (n *noLeaderElection) Stop() error {
	return nil
}

func (n *noLeaderElection) IsLeader() bool {
	return true
}
