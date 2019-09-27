package test

import (
	"context"
	"go.etcd.io/etcd/clientv3"
)

type FakeETCDLease struct{}

func (*FakeETCDLease) Grant(ctx context.Context, ttl int64) (*clientv3.LeaseGrantResponse, error) {
	return &clientv3.LeaseGrantResponse{ID: 100}, nil
}

func (*FakeETCDLease) Revoke(ctx context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) {
	panic("implement me")
}

func (*FakeETCDLease) TimeToLive(ctx context.Context, id clientv3.LeaseID, opts ...clientv3.LeaseOption) (*clientv3.LeaseTimeToLiveResponse, error) {
	panic("implement me")
}

func (*FakeETCDLease) Leases(ctx context.Context) (*clientv3.LeaseLeasesResponse, error) {
	panic("implement me")
}

func (*FakeETCDLease) KeepAlive(ctx context.Context, id clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	panic("implement me")
}

func (*FakeETCDLease) KeepAliveOnce(ctx context.Context, id clientv3.LeaseID) (*clientv3.LeaseKeepAliveResponse, error) {
	panic("implement me")
}

func (*FakeETCDLease) Close() error {
	panic("implement me")
}
