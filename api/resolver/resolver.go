package resolver

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/TrueCloudLab/frostfs-sdk-go/container"
	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/ns"
)

const (
	NNSResolver = "nns"
	DNSResolver = "dns"
)

// ErrNoResolvers returns when trying to resolve container without any resolver.
var ErrNoResolvers = errors.New("no resolvers")

// FrostFS represents virtual connection to the FrostFS network.
type FrostFS interface {
	// SystemDNS reads system DNS network parameters of the FrostFS.
	//
	// It returns exactly on non-zero value. It returns any error encountered
	// which prevented the parameter from being read.
	SystemDNS(context.Context) (string, error)
}

type Config struct {
	FrostFS    FrostFS
	RPCAddress string
}

type BucketResolver struct {
	mu        sync.RWMutex
	resolvers []*Resolver
}

type Resolver struct {
	Name    string
	resolve func(context.Context, string) (cid.ID, error)
}

func (r *Resolver) SetResolveFunc(fn func(context.Context, string) (cid.ID, error)) {
	r.resolve = fn
}

func (r *Resolver) Resolve(ctx context.Context, name string) (cid.ID, error) {
	return r.resolve(ctx, name)
}

func NewBucketResolver(resolverNames []string, cfg *Config) (*BucketResolver, error) {
	resolvers, err := createResolvers(resolverNames, cfg)
	if err != nil {
		return nil, err
	}

	return &BucketResolver{
		resolvers: resolvers,
	}, nil
}

func createResolvers(resolverNames []string, cfg *Config) ([]*Resolver, error) {
	resolvers := make([]*Resolver, len(resolverNames))
	for i, name := range resolverNames {
		cnrResolver, err := newResolver(name, cfg)
		if err != nil {
			return nil, err
		}
		resolvers[i] = cnrResolver
	}

	return resolvers, nil
}

func (r *BucketResolver) Resolve(ctx context.Context, bktName string) (cnrID cid.ID, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, resolver := range r.resolvers {
		cnrID, resolverErr := resolver.Resolve(ctx, bktName)
		if resolverErr != nil {
			resolverErr = fmt.Errorf("%s: %w", resolver.Name, resolverErr)
			if err == nil {
				err = resolverErr
			} else {
				err = fmt.Errorf("%s: %w", err.Error(), resolverErr)
			}
			continue
		}
		return cnrID, nil
	}

	if err != nil {
		return cnrID, err
	}

	return cnrID, ErrNoResolvers
}

func (r *BucketResolver) UpdateResolvers(resolverNames []string, cfg *Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.equals(resolverNames) {
		return nil
	}

	resolvers, err := createResolvers(resolverNames, cfg)
	if err != nil {
		return err
	}

	r.resolvers = resolvers

	return nil
}

func (r *BucketResolver) equals(resolverNames []string) bool {
	if len(r.resolvers) != len(resolverNames) {
		return false
	}

	for i := 0; i < len(resolverNames); i++ {
		if r.resolvers[i].Name != resolverNames[i] {
			return false
		}
	}
	return true
}

func newResolver(name string, cfg *Config) (*Resolver, error) {
	switch name {
	case DNSResolver:
		return NewDNSResolver(cfg.FrostFS)
	case NNSResolver:
		return NewNNSResolver(cfg.RPCAddress)
	default:
		return nil, fmt.Errorf("unknown resolver: %s", name)
	}
}

func NewDNSResolver(frostFS FrostFS) (*Resolver, error) {
	if frostFS == nil {
		return nil, fmt.Errorf("pool must not be nil for DNS resolver")
	}

	var dns ns.DNS

	resolveFunc := func(ctx context.Context, name string) (cid.ID, error) {
		domain, err := frostFS.SystemDNS(ctx)
		if err != nil {
			return cid.ID{}, fmt.Errorf("read system DNS parameter of the FrostFS: %w", err)
		}

		domain = name + "." + domain
		cnrID, err := dns.ResolveContainerName(domain)
		if err != nil {
			return cid.ID{}, fmt.Errorf("couldn't resolve container '%s' as '%s': %w", name, domain, err)
		}
		return cnrID, nil
	}

	return &Resolver{
		Name:    DNSResolver,
		resolve: resolveFunc,
	}, nil
}

func NewNNSResolver(address string) (*Resolver, error) {
	if address == "" {
		return nil, fmt.Errorf("rpc address must not be empty for NNS resolver")
	}

	var nns ns.NNS

	if err := nns.Dial(address); err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}

	resolveFunc := func(_ context.Context, name string) (cid.ID, error) {
		var d container.Domain
		d.SetName(name)

		cnrID, err := nns.ResolveContainerDomain(d)
		if err != nil {
			return cid.ID{}, fmt.Errorf("couldn't resolve container '%s': %w", name, err)
		}
		return cnrID, nil
	}

	return &Resolver{
		Name:    NNSResolver,
		resolve: resolveFunc,
	}, nil
}
