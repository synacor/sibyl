package deck

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeck(t *testing.T) {
	d := &Deck{"Test", []string{"A", "B", "C"}}

	c, err := d.GetCard(0)
	assert.Equal(t, "A", c)
	assert.NoError(t, err)

	c, err = d.GetCard(2)
	assert.Equal(t, "C", c)
	assert.NoError(t, err)

	c, err = d.GetCard(3)
	assert.Equal(t, "", c)
	assert.Error(t, err)
	assert.Equal(t, err, ErrCardNotFound)
}

func TestSpotCheck(t *testing.T) {
	c, _ := ModifiedFibonacci.GetCard(7)
	assert.Equal(t, "20", c)

	c, _ = Fibonacci.GetCard(7)
	assert.Equal(t, "21", c)

	c, _ = TShirtSizes.GetCard(0)
	assert.Equal(t, "XS", c)

	c, _ = Hours.GetCard(1)
	assert.Equal(t, ".5", c)
}

func TestAllDecks(t *testing.T) {
	for k, d := range AllDecks {
		assert.Equal(t, k, d.Name)
	}
}
