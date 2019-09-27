package storage

import (
	"context"
	"errors"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"strings"
	"time"
)

type LightningMonkeyETCDStorageDriver struct {
	client         *clientv3.Client
	requestTimeout time.Duration
}

func (sd *LightningMonkeyETCDStorageDriver) GetRequestTimeoutDuration() time.Duration {
	return sd.requestTimeout
}

//Required Fields:
// + ENDPOINTS
func (sd *LightningMonkeyETCDStorageDriver) Initialize(settings map[string]string) error {
	//inject default values.
	if settings["DIAL_TIMEOUT"] == "" {
		settings["DIAL_TIMEOUT"] = "5s"
	}
	if settings["REQUEST_TIMEOUT"] == "" {
		settings["REQUEST_TIMEOUT"] = "5s"
	}
	//do initialization.
	dialTimeout, err := time.ParseDuration(settings["DIAL_TIMEOUT"])
	if err != nil {
		return fmt.Errorf("Failed to parse required argument: \"DIAL_TIMEOUT\", error: %s", err.Error())
	}
	sd.requestTimeout, err = time.ParseDuration(settings["REQUEST_TIMEOUT"])
	if err != nil {
		return fmt.Errorf("Failed to parse required argument: \"REQUEST_TIMEOUT\", error: %s", err.Error())
	}
	if settings["ENDPOINTS"] == "" {
		return errors.New("Argument \"ENDPOINTS\" is required for initializing ETCD client!")
	}
	config := clientv3.Config{
		Endpoints:   strings.Split(settings["ENDPOINTS"], ","),
		DialTimeout: dialTimeout,
	}
	sd.client, err = clientv3.New(config)
	if err != nil {
		return fmt.Errorf("Failed to initialize ETCD v3 client, error: %s", err.Error())
	}
	return nil
}

func (sd *LightningMonkeyETCDStorageDriver) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	return sd.client.Get(ctx, key, opts...)
}

func (sd *LightningMonkeyETCDStorageDriver) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	return sd.client.Watch(ctx, key, opts...)
}

func (sd *LightningMonkeyETCDStorageDriver) Txn(ctx context.Context) clientv3.Txn {
	return sd.client.Txn(ctx)
}

func (sd *LightningMonkeyETCDStorageDriver) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	return sd.client.Put(ctx, key, val, opts...)

}

func (sd *LightningMonkeyETCDStorageDriver) NewLease() clientv3.Lease {
	return clientv3.NewLease(sd.client)
}
