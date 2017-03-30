package name

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	// manually set the seed so we get a consistent result
	rnd = rand.New(rand.NewSource(3))

	assert.Equal(t, "Chilly Butterfly", Generate())
	assert.Equal(t, "Petite Penguin", Generate())
}
