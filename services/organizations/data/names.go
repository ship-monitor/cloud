package data

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

var (
	//nolint:gochecknoglobals
	Adjectives = []string{
		"Cool",
		"Splendid",
		"Awesome",
		"Different",
		"Soft",
		"Good",
		"Happy",
		"Old",
		"Great",
		"New",
		"Big",
		"Small",
		"Tall",
		"Short",
		"Long",
		"Wide",
		"High",
	}
	//nolint:gochecknoglobals
	Nouns = []string{
		"Node", "Thing", "Box", "Service", "Child", "Line", "Statement",
		"Flower", "Cat", "Sheep",
	}
)

//nolint:gochecknoglobals
var defaultName = fmt.Sprintf("%s %s", Adjectives[0], Nouns[0])

func GenNodeName(id uuid.UUID) string {
	adj, err := rand.Int(rand.Reader, big.NewInt(int64(len(Adjectives))))
	if err != nil {
		return defaultName
	}

	noun, err := rand.Int(rand.Reader, big.NewInt(int64(len(Nouns))))
	if err != nil {
		return defaultName
	}

	return fmt.Sprintf("%s %s", Adjectives[adj.Int64()], Nouns[noun.Int64()])
}
