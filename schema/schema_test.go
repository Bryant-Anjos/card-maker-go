package schema_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/schema"
)

// baseURL must match the $id prefix declared inside the schema files.
const baseURL = "https://card-maker-go/schema/"

type contract struct {
	name   string
	schema string
	parse  func(src string) error
	files  []string
}

var contracts = []contract{
	{
		name:   "cards",
		schema: "cards.schema.json",
		parse:  func(src string) error { _, err := model.ParseCardSet(strings.NewReader(src)); return err },
		files:  []string{"../testdata/cards/cards.json", "../testdata/cards/variants.json", "../testdata/cards/longnames.json"},
	},
	{
		name:   "theme",
		schema: "theme.schema.json",
		parse:  func(src string) error { _, err := model.ParseTheme(strings.NewReader(src)); return err },
		files:  []string{"../themes/default/theme.json"},
	},
	{
		name:   "manifest",
		schema: "manifest.schema.json",
		parse:  func(src string) error { _, err := model.ParseManifest(strings.NewReader(src)); return err },
		files:  []string{"../testdata/manifest/manifest.json"},
	},
}

func compile(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()

	compiler := jsonschema.NewCompiler()
	for _, file := range []string{"defs.schema.json", name} {
		raw, err := schema.FS.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(raw)))
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		if err := compiler.AddResource(baseURL+file, doc); err != nil {
			t.Fatalf("add %s: %v", file, err)
		}
	}

	compiled, err := compiler.Compile(baseURL + name)
	if err != nil {
		t.Fatalf("compile %s: %v", name, err)
	}

	return compiled
}

func validatesAgainst(t *testing.T, compiled *jsonschema.Schema, src string) error {
	t.Helper()

	var doc any
	if err := json.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	return compiled.Validate(doc)
}

// Valid sample files must be accepted by both the JSON Schema and the Go model,
// so the two definitions of the contract cannot drift apart silently.
func TestSampleFilesConformToSchemaAndModel(t *testing.T) {
	for _, c := range contracts {
		compiled := compile(t, c.schema)

		for _, file := range c.files {
			t.Run(c.name+" "+file, func(t *testing.T) {
				raw, err := os.ReadFile(file)
				if err != nil {
					t.Fatalf("read sample: %v", err)
				}

				if err := validatesAgainst(t, compiled, string(raw)); err != nil {
					t.Errorf("schema rejected sample: %v", err)
				}
				if err := c.parse(string(raw)); err != nil {
					t.Errorf("model rejected sample: %v", err)
				}
			})
		}
	}
}

// Invalid documents must be rejected by both sides.
func TestInvalidDocumentsRejectedByBoth(t *testing.T) {
	const card = `{"id":"a","rarity":"common","name":{"pt-BR":"A"},"art":"a.png"}`

	tests := []struct {
		name string
		c    contract
		src  string
	}{
		{"cards unknown field", contracts[0], `{"schemaVersion":1,"cards":[` + card + `],"x":1}`},
		{"cards wrong version", contracts[0], `{"schemaVersion":2,"cards":[` + card + `]}`},
		{"cards no cards", contracts[0], `{"schemaVersion":1,"cards":[]}`},
		{"cards uppercase id", contracts[0], `{"schemaVersion":1,"cards":[{"id":"A","rarity":"c","name":{"pt-BR":"A"},"art":"a.png"}]}`},
		{"cards art escapes root", contracts[0], `{"schemaVersion":1,"cards":[{"id":"a","rarity":"c","name":{"pt-BR":"A"},"art":"../../x.png"}]}`},
		{"cards art absolute", contracts[0], `{"schemaVersion":1,"cards":[{"id":"a","rarity":"c","name":{"pt-BR":"A"},"art":"/x.png"}]}`},
		{"cards art dot segment", contracts[0], `{"schemaVersion":1,"cards":[{"id":"a","rarity":"c","name":{"pt-BR":"A"},"art":"a/./x.png"}]}`},
		{"cards art empty segment", contracts[0], `{"schemaVersion":1,"cards":[{"id":"a","rarity":"c","name":{"pt-BR":"A"},"art":"a//x.png"}]}`},
		{"cards art trailing slash", contracts[0], `{"schemaVersion":1,"cards":[{"id":"a","rarity":"c","name":{"pt-BR":"A"},"art":"a/"}]}`},
		{"cards art backslash", contracts[0], `{"schemaVersion":1,"cards":[{"id":"a","rarity":"c","name":{"pt-BR":"A"},"art":"a\\x.png"}]}`},
		{"cards missing name", contracts[0], `{"schemaVersion":1,"cards":[{"id":"a","rarity":"c","art":"a.png"}]}`},
		{"cards bad locale", contracts[0], `{"schemaVersion":1,"cards":[{"id":"a","rarity":"c","name":{"português":"A"},"art":"a.png"}]}`},
		{"cards bad variant", contracts[0], `{"schemaVersion":1,"cards":[{"id":"a","rarity":"c","variant":"Full Art","name":{"pt-BR":"A"},"art":"a.png"}]}`},
		{"theme variant without template", contracts[1], `{"schemaVersion":1,"name":"t","size":{"width":1,"height":1},"template":"a","stylesheet":"b","rarities":[{"id":"c","color":"#ffffff","frame":"f"}],"variants":[{"id":"fullart"}]}`},
		{"theme variant bad id", contracts[1], `{"schemaVersion":1,"name":"t","size":{"width":1,"height":1},"template":"a","stylesheet":"b","rarities":[{"id":"c","color":"#ffffff","frame":"f"}],"variants":[{"id":"Full","template":"x"}]}`},
		{"manifest bad variant", contracts[2], `{"schemaVersion":1,"theme":"t","locale":"pt-BR","size":{"width":1,"height":1},"cards":[{"id":"a","rarity":"c","variant":"X Y","name":"A","images":[{"scale":1,"format":"png","path":"a.png","width":1,"height":1}]}]}`},
		{"theme bad color", contracts[1], `{"schemaVersion":1,"name":"t","size":{"width":1,"height":1},"template":"a","stylesheet":"b","rarities":[{"id":"c","color":"gray","frame":"f"}]}`},
		{"theme zero width", contracts[1], `{"schemaVersion":1,"name":"t","size":{"width":0,"height":1},"template":"a","stylesheet":"b","rarities":[{"id":"c","color":"#ffffff","frame":"f"}]}`},
		{"theme no rarities", contracts[1], `{"schemaVersion":1,"name":"t","size":{"width":1,"height":1},"template":"a","stylesheet":"b","rarities":[]}`},
		{"manifest bad format", contracts[2], `{"schemaVersion":1,"theme":"t","locale":"pt-BR","size":{"width":1,"height":1},"cards":[{"id":"a","rarity":"c","name":"A","images":[{"scale":1,"format":"gif","path":"a.gif","width":1,"height":1}]}]}`},
		{"manifest no images", contracts[2], `{"schemaVersion":1,"theme":"t","locale":"pt-BR","size":{"width":1,"height":1},"cards":[{"id":"a","rarity":"c","name":"A","images":[]}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled := compile(t, tt.c.schema)

			if err := validatesAgainst(t, compiled, tt.src); err == nil {
				t.Error("schema accepted an invalid document")
			}
			if err := tt.c.parse(tt.src); err == nil {
				t.Error("model accepted an invalid document")
			}
		})
	}
}
