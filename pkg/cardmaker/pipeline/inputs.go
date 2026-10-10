package pipeline

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // register decoders so artwork formats can be checked
	_ "image/png"
	"os"
	"path/filepath"

	_ "golang.org/x/image/webp"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/theme"
)

// cardInput is a card ready to render: texts resolved, artwork located.
type cardInput struct {
	card    i18n.ResolvedCard
	artPath string
}

// inputs is everything a build needs, fully validated.
type inputs struct {
	theme     theme.Theme
	themeHash string
	cards     []cardInput
}

// loadInputs loads and validates the theme, the cards data and the artwork.
// Nothing is rendered or written, so every problem surfaces before any work.
func loadInputs(opts Options) (inputs, error) {
	loadedTheme, err := theme.LoadDir(opts.ThemeDir)
	if err != nil {
		return inputs{}, fmt.Errorf("theme %q: %w", opts.ThemeDir, err)
	}

	set, err := readCardSet(opts.CardsFile)
	if err != nil {
		return inputs{}, err
	}
	if err := set.ValidateAgainst(loadedTheme.Theme); err != nil {
		return inputs{}, fmt.Errorf("cards file %q:\n%w", opts.CardsFile, err)
	}

	resolver, err := i18n.NewResolver(opts.Locale, opts.Fallback)
	if err != nil {
		return inputs{}, err
	}
	resolved, err := resolver.Cards(set.Cards)
	if err != nil {
		return inputs{}, err
	}

	cards, err := locateArt(resolved, filepath.Dir(opts.CardsFile))
	if err != nil {
		return inputs{}, err
	}

	themeHash, err := hashFS(loadedTheme.FS)
	if err != nil {
		return inputs{}, fmt.Errorf("hash theme: %w", err)
	}

	return inputs{theme: loadedTheme, themeHash: themeHash, cards: cards}, nil
}

func readCardSet(path string) (model.CardSet, error) {
	file, err := os.Open(path)
	if err != nil {
		return model.CardSet{}, fmt.Errorf("cards file: %w", err)
	}
	defer file.Close()

	set, err := model.ParseCardSet(file)
	if err != nil {
		return model.CardSet{}, fmt.Errorf("cards file %q: %w", path, err)
	}

	return set, nil
}

// locateArt checks that every artwork file exists and is a supported image,
// reporting all problems together.
func locateArt(cards []i18n.ResolvedCard, baseDir string) ([]cardInput, error) {
	inputs := make([]cardInput, 0, len(cards))

	var errs []error
	for _, card := range cards {
		path := filepath.Join(baseDir, filepath.FromSlash(card.Art))
		if err := checkImageFile(path); err != nil {
			errs = append(errs, fmt.Errorf("card %q: art %q: %w", card.ID, card.Art, err))
			continue
		}
		inputs = append(inputs, cardInput{card: card, artPath: path})
	}

	return inputs, errors.Join(errs...)
}

func checkImageFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, _, err := image.DecodeConfig(file); err != nil {
		return fmt.Errorf("is not a supported image (png, jpeg or webp): %w", err)
	}

	return nil
}
