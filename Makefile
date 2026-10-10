.PHONY: build test vet cover check testdata-art theme-frames golden

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
