package model_test

import (
	"fmt"
	"strings"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
)

// A Go program can consume a manifest with the same types the pipeline uses.
func ExampleParseManifest() {
	src := `{"schemaVersion":1,"theme":"default","locale":"pt-BR","size":{"width":750,"height":1050},
		"cards":[{"id":"vulpin","rarity":"common","name":"Vulpin",
		"images":[{"scale":1,"format":"png","path":"cards/vulpin.png","width":750,"height":1050}]}]}`

	manifest, err := model.ParseManifest(strings.NewReader(src))
	if err != nil {
		panic(err)
	}

	for _, card := range manifest.Cards {
		fmt.Println(card.Name, card.Images[0].Path)
	}
	// Output: Vulpin cards/vulpin.png
}
