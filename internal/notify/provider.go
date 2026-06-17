package notify

import (
	"context"
	"fmt"
	"sort"
)

type Config map[string]string

type Provider interface {
	Send(ctx context.Context, config Config, message string) error
	ValidateConfig(config Config) error
	SecretKeys() []string
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(providers map[string]Provider) *Registry {
	return &Registry{providers: providers}
}

func (r *Registry) Send(ctx context.Context, provider string, config map[string]string, message string) error {
	p, ok := r.providers[provider]
	if !ok {
		return fmt.Errorf("unknown provider: %s", provider)
	}

	return p.Send(ctx, Config(config), message)
}

func (r *Registry) ValidateConfig(provider string, config map[string]string) error {
	p, ok := r.providers[provider]
	if !ok {
		return fmt.Errorf("unknown provider %q", provider)
	}

	return p.ValidateConfig(Config(config))
}

func (r *Registry) SecretKeys(provider string) []string {
	p, ok := r.providers[provider]
	if !ok {
		return nil
	}

	return p.SecretKeys()
}

func (r *Registry) KnownProviders() []string {
	keys := make([]string, 0, len(r.providers))
	for k := range r.providers {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}
