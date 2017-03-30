// Package name generates random names
package name

import (
	"math/rand"
	"strings"
	"time"
)

var adjectives = strings.Split("Able|Abundant|Adorable|Agreeable|Ancient|Angry|Bad|Beautiful|Better|Bewildered|Big|Bitter|Black|Blue|Boiling|Brave|Breeze|Brief|Broad|Broken|Bumpy|Calm|Careful|Chilly|Chubby|Clean|Clever|Clumsy|Cold|Cool|Creepy|Crooked|Cuddly|Curly|Curved|Damaged|Damp|Dead|Deafening|Deep|Defeated|Delicious|Delightful|Different|Drab|Dry|Eager|Early|Easy|Elegant|Embarrassed|Empty|Faithful|Famous|Fancy|Fast|Fat|Fierce|Filthy|First|Flaky|Flat|Fluffy|Freezing|Fresh|Full|Gentle|Gifted|Gigantic|Glamorous|Good|Gray|Greasy|Great|Green|Grumpy|Happy|Heavy|Helpful|Helpless|High|Hissing|Hollow|Huge|Icy|Important|Jealous|Jolly|Kind|Large|Last|Late|Lazy|Light|Little|Lively|Long|Loud|Low|Magnificent|Mammoth|Many|Massive|Melodic|Melted|Miniature|Modern|Mushy|Mysterious|Narrow|Nervous|New|Next|Nice|Noisy|Numerous|Obedient|Obnoxious|Odd|Old|Orange|Own|Panicky|Petite|Plain|Powerful|Prickly|Proud|Public|Puny|Purple|Purring|Quaint|Quick|Quiet|Rainy|Rapid|Raspy|Red|Relieved|Repulsive|Rich|Right|Rotten|Round|Salty|Same|Scary|Scrawny|Screeching|Shallow|Short|Shy|Silly|Slow|Small|Sparkling|Sparse|Square|Steep|Sticky|Straight|Strong|Substantial|Sweet|Swift|Tall|Tasteless|Thankful|Thoughtless|Thundering|Tiny|Ugliest|Uneven|Uninterested|Unsightly|Uptight|Vast|Victorious|Voiceless|Warm|Weak|Whispering|White|Wide|Witty|Wooden|Worried|Wrong|Yellow|Young|Yummy|Zealous", "|")

var animals = strings.Split("Albatross|Alligator|Anteater|Antelope|Armadillo|Baboon|Badger|Bandicoot|Barracuda|Bat|Bear|Bird|Bison|Bobcat|Bonobo|Buffalo|Bullfrog|Butterfly|Camel|Capybara|Cat|Caterpillar|Catfish|Chameleon|Cheetah|Chicken|Chimpanzee|Chinchilla|Chipmunk|Cougar|Cow|Coyote|Crab|Crocodile|Deer|Dingo|Dog|Dolphin|Donkey|Duck|Eagle|Elephant|Emu|Falcon|Ferret|Flamingo|Fox|Frog|Gecko|Gerbil|Gharial|Giraffe|Goat|Goose|Gopher|Gorilla|Hamster|Hare|Hedgehog|Horse|Jackal|Jaguar|Kangaroo|Kiwi|Koala|Lemming|Lemur|Leopard|Liger|Lion|Lizard|Llama|Lobster|Mandrill|Meerkat|Mongoose|Mongrel|Monkey|Moose|Mouse|Mule|Ocelot|Octopus|Opossum|Ostrich|Otter|Panther|Parrot|Peacock|Pelican|Penguin|Pig|Platypus|Possum|Rabbit|Raccoon|Rat|Rattlesnake|Reindeer|Rhinoceros|Salamander|Scorpion|Seahorse|Seal|Serval|Sheep|Shrimp|Skunk|Sloth|Snake|Squid|Squirrel|Starfish|Stingray|Tapir|Tiger|Tortoise|Toucan|Turkey|Vulture|Wallaby|Walrus|Warthog|Wasp|Weasel|Wildebeest|Wolf|Wolverine|Wombat|Woodpecker|Yak|Zebra", "|")

// no need for crypto/rand, so we'll seed with a timestamp so we can easily test
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

// Generate will return a random name
func Generate() string {
	return randomWord(adjectives) + " " + randomWord(animals)
}

func randomWord(list []string) string {
	return list[rnd.Intn(len(list))]
}
