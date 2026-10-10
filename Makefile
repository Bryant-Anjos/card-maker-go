.PHONY: build test vet cover check testdata-art theme-frames golden docs-images docs-check docs-site

build:
	go build -o bin/cardmaker ./cmd/cardmaker
	go build -o bin/cardmaker-server ./cmd/cardmaker-server

test:
	go test ./...

vet:
	go vet ./...

cover:
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

check: vet test

# Regenerates the placeholder artwork referenced by testdata/cards/cards.json.
testdata-art:
	go run ./internal/devtools/placeholderart

# Regenerates the per-rarity ornaments of the default theme.
theme-frames:
	python3 internal/devtools/framegen/gen_frames.py themes/default/frames

# Rewrites the golden images after an intentional visual change. Review the
# images (and testdata/golden/default/RENDERER) before committing.
golden:
	go test ./pkg/cardmaker/render/chrome/ -run Golden -update -count=1

# Re-renders the README/docs illustrations with the real tool (needs a Chromium
# browser) and records the fingerprint of the sources they came from.
docs-images:
	python3 internal/devtools/docimages/make_images.py
	go run ./internal/devtools/docsourcehash > docs/images/SOURCE_HASH

# Fails when the documentation drifted from the code.
docs-check:
	go test -count=1 ./internal/docscheck/

# Monta o site da documentação em site/ como o GitHub Pages faz (precisa de
# `pip install -r requirements-docs.txt`). --strict falha com link quebrado.
docs-site:
	python3 -m mkdocs build --strict
