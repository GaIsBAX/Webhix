package notify

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type Config map[string]string

type Provider interface {
	Send(ctx context.Context, config Config, message string) error
	ValidateConfig(config Config) error
	SecretKeys() []string
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Provider{
		"telegram": telegramProvider{},
	}
)

func Send(ctx context.Context, provider string, config Config, message string) error {
	registryMu.RLock()

	p, ok := registry[provider]
	registryMu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown provider: %s", provider)
	}

	return p.Send(ctx, config, message)
}

func ValidateConfig(provider string, config Config) error {
	registryMu.RLock()
	p, ok := registry[provider]
	registryMu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown provider %q", provider)
	}

	return p.ValidateConfig(config)
}

func SecretKeys(provider string) []string {
	registryMu.RLock()
	p, ok := registry[provider]
	registryMu.RUnlock()

	if !ok {
		return nil
	}

	return p.SecretKeys()
}

func KnownProviders() []string {
	registryMu.RLock()

	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}

	registryMu.RUnlock()
	sort.Strings(keys)
	return keys
}
