package connections

import (
	"fmt"
	"math/rand/v2"
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

func GenNodeName() string {
	adj := rand.IntN(len(Adjectives))
	noun := rand.IntN(len(Nouns))
	return fmt.Sprintf("%s %s", Adjectives[adj], Nouns[noun])
}
