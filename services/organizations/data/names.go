package data

import (
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"
)

var (
	Adjectives = []string{
		"Cool", "Splendid", "Awesome", "Different", "Soft",
		"Good", "Happy", "Old", "Great", "New", "Big", "Small", "Tall", "Short", "Long", "Wide", "High",
	}
	Nouns = []string{
		"Node", "Thing", "Box", "Service", "Child", "Line", "Statement",
		"Flower", "Cat", "Sheep",
	}
)

type source struct {
	seed uuid.UUID
}

func (s *source) Uint64() uint64 {
	return uint64(s.seed.ID())
}

func GenNodeName(id uuid.UUID) string {
	generator := rand.New(&source{seed: id})
	adj := generator.IntN(len(Adjectives))
	noun := generator.IntN(len(Nouns))
	return fmt.Sprintf("%s %s", Adjectives[adj], Nouns[noun])
}
