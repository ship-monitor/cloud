package subscriber

import (
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
)

type subsContainer struct {
	subs []*nats.Subscription
}

func newSubsContainer() *subsContainer {
	return &subsContainer{
		subs: []*nats.Subscription{},
	}
}

func (c *subsContainer) Add(s *nats.Subscription) {
	c.subs = append(c.subs, s)
}

func (c *subsContainer) Unsubscribe() error {
	var err error
	for _, s := range c.subs {
		err = errors.Join(err, s.Unsubscribe())
	}

	if err != nil {
		return fmt.Errorf("unsubscribe all: %w", err)
	}

	return nil
}
