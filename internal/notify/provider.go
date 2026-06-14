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
}

type ProviderFunc func(ctx context.Context, config Config, message string) error

func (f ProviderFunc) Send(ctx context.Context, config Config, message string) error {
	return f(ctx, config, message)
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Provider{
		"telegram": ProviderFunc(telegramSend),
	}

	secretKeys = map[string][]string{
		"telegram": {"bot_token"},
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

func SecretKeys(provider string) []string {
	return secretKeys[provider]
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
