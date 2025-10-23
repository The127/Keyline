package quorum

import (
	"Keyline/internal/config"
	"Keyline/internal/logging"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/hashicorp/raft"
)

type raftLeaderElection struct {
	raft     *raft.Raft
	callback LeaderChangeCallback
	isLeader bool
}

func NewRaftLeaderElection(callback LeaderChangeCallback) LeaderElection {
	return &raftLeaderElection{
		callback: callback,
	}
}

func (r *raftLeaderElection) Start(ctx context.Context) error {
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(config.C.LeaderElection.Raft.Id)

	store := raft.NewInmemStore()
	snapshots := raft.NewInmemSnapshotStore()

	bindAddr := fmt.Sprintf("%s:%d", config.C.LeaderElection.Raft.Host, config.C.LeaderElection.Raft.Port)
	advertise, err := net.ResolveTCPAddr("tcp", bindAddr)
	if err != nil {
		return fmt.Errorf("resolve TCP addr: %w", err)
	}

	transport, err := raft.NewTCPTransport(bindAddr, advertise, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return fmt.Errorf("transport: %w", err)
	}

	r.raft, err = raft.NewRaft(raftConfig, &noopFSM{}, store, store, snapshots, transport)
	if err != nil {
		return fmt.Errorf("new raft: %w", err)
	}

	if config.C.LeaderElection.Raft.Id == config.C.LeaderElection.Raft.InitiatorId {
		servers := make([]raft.Server, len(config.C.LeaderElection.Raft.Nodes))
		for i, node := range config.C.LeaderElection.Raft.Nodes {
			servers[i] = raft.Server{
				ID:      raft.ServerID(node.Id),
				Address: raft.ServerAddress(node.Address),
			}
		}

		cfg := raft.Configuration{Servers: servers}
		future := r.raft.BootstrapCluster(cfg)
		if err := future.Error(); err != nil {
			return fmt.Errorf("bootstrap: %w", err)
		}

		logging.Logger.Infof("Bootstrapped Raft cluster with %d nodes", len(servers))
	}

	go r.watchLeadership(ctx)
	return nil
}

func (r *raftLeaderElection) watchLeadership(ctx context.Context) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	prev := raft.Follower
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state := r.raft.State()
			if state != prev {
				logging.Logger.Infof("Raft leader changed to %s", state)
				r.isLeader = state == raft.Leader
				go r.callback(r.isLeader)
				prev = state
			}
		}
	}
}

func (r *raftLeaderElection) Stop() error {
	if r.raft != nil {
		f := r.raft.Shutdown()
		return f.Error()
	}
	return nil
}

func (r *raftLeaderElection) IsLeader() bool {
	return r.isLeader
}

type noopFSM struct{}

func (n *noopFSM) Apply(*raft.Log) interface{}         { return nil }
func (n *noopFSM) Snapshot() (raft.FSMSnapshot, error) { return &noopSnapshot{}, nil }
func (n *noopFSM) Restore(io.ReadCloser) error         { return nil }

type noopSnapshot struct{}

func (n *noopSnapshot) Persist(sink raft.SnapshotSink) error {
	err := sink.Close()
	if err != nil {
		return fmt.Errorf("persist: %w", err)
	}
	return nil
}
func (n *noopSnapshot) Release() {}
