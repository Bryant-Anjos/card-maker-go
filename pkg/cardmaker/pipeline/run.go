package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
)

const (
	manifestFileName = "manifest.json"
	cardsDir         = "cards"
	backBaseName     = "back"
)

// Result summarizes a build. Counts are in rendered images (one per card and
// scale); formats are encoded from the same render and are not counted.
type Result struct {
	OutDir   string // directory holding the manifest and images
	Manifest model.Manifest
	Rendered int
	Skipped  int
}

// Validate loads and checks every input without rendering or writing anything,
// and returns how many cards the build would produce. OutDir is not needed.
func Validate(opts Options) (int, error) {
	normalized, err := opts.normalized()
	if err != nil {
		return 0, err
	}

	in, err := loadInputs(normalized)
	if err != nil {
		return 0, err
	}

	return len(in.cards), nil
}

// Run builds the collection into OutDir/<locale>/. Inputs are fully validated
// before the renderer is touched, and outputs that are already up to date are
// skipped unless Force is set.
func Run(ctx context.Context, opts Options, renderer render.Renderer) (Result, error) {
	normalized, err := opts.normalized()
	if err != nil {
		return Result{}, err
	}
	if normalized.OutDir == "" {
		return Result{}, errors.New("output directory (out) is required")
	}
	in, err := loadInputs(normalized)
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	b := &build{
		opts:     normalized,
		in:       in,
		renderer: renderer,
		outDir:   filepath.Join(normalized.OutDir, normalized.Locale),
	}
	b.cache = loadCache(filepath.Join(b.outDir, cacheFileName))

	return b.run(ctx)
}

// build holds the state shared by the tasks of one Run.
type build struct {
	opts     Options
	in       inputs
	renderer render.Renderer
	outDir   string
	cache    *cache

	rendered atomic.Int64
	skipped  atomic.Int64
}

func (b *build) run(ctx context.Context) (Result, error) {
	cardImages := make([][]model.Image, len(b.in.cards))
	var backImages []model.Image

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(b.opts.Concurrency)

	for i := range b.in.cards {
		group.Go(func() error {
			images, err := b.buildCard(groupCtx, b.in.cards[i])
			cardImages[i] = images

			return err
		})
	}
	if b.in.theme.Back != "" {
		group.Go(func() error {
			images, err := b.buildBack(groupCtx)
			backImages = images

			return err
		})
	}

	runErr := group.Wait()

	// Save progress even after a failure so finished work is not repeated.
	saveErr := b.cache.save(filepath.Join(b.outDir, cacheFileName))
	if runErr != nil {
		return Result{}, runErr
	}
	if saveErr != nil {
		return Result{}, fmt.Errorf("save cache: %w", saveErr)
	}

	manifest := b.manifest(cardImages, backImages)
	if err := writeManifest(filepath.Join(b.outDir, manifestFileName), manifest); err != nil {
		return Result{}, err
	}

	return Result{
		OutDir:   b.outDir,
		Manifest: manifest,
		Rendered: int(b.rendered.Load()),
		Skipped:  int(b.skipped.Load()),
	}, nil
}

func (b *build) manifest(cardImages [][]model.Image, backImages []model.Image) model.Manifest {
	cards := make([]model.ManifestCard, len(b.in.cards))
	for i, input := range b.in.cards {
		cards[i] = model.ManifestCard{
			ID:          input.card.ID,
			Number:      input.card.Number,
			Rarity:      input.card.Rarity,
			Element:     input.card.Element,
			Name:        input.card.Name,
			Description: input.card.Description,
			Images:      cardImages[i],
		}
	}

	return model.Manifest{
		SchemaVersion: model.SchemaVersion,
		Theme:         b.in.theme.Name,
		Locale:        b.opts.Locale,
		Size:          b.in.theme.Size,
		Back:          backImages,
		Cards:         cards,
	}
}

func writeManifest(path string, manifest model.Manifest) error {
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("generated manifest is invalid: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(path, append(data, '\n'))
}

// plannedImages lists the files a base name produces: one per scale and format.
func (b *build) plannedImages(dir, base string) []model.Image {
	size := b.in.theme.Size

	var images []model.Image
	for _, scale := range b.opts.Scales {
		for _, format := range b.opts.Formats {
			images = append(images, model.Image{
				Scale:  scale,
				Format: format,
				Path:   imagePath(dir, imageName(base, scale, format)),
				Width:  size.Width * scale,
				Height: size.Height * scale,
			})
		}
	}

	return images
}

func (b *build) buildCard(ctx context.Context, input cardInput) ([]model.Image, error) {
	card := input.card
	planned := b.plannedImages(cardsDir, card.ID)

	art, err := os.ReadFile(input.artPath)
	if err != nil {
		return nil, fmt.Errorf("card %q: read art: %w", card.ID, err)
	}
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return nil, err
	}

	cacheName := "card:" + card.ID
	key := keyOf(version, b.renderer.Version(), b.in.themeHash, string(cardJSON), hashBytes(art),
		joinInts(b.opts.Scales), fmt.Sprint(b.opts.Formats))

	if b.upToDate(cacheName, key, planned) {
		b.skipped.Add(int64(len(b.opts.Scales)))

		return planned, nil
	}

	renderScale := func(scale int) ([]byte, error) {
		return b.renderer.RenderCard(ctx, renderCardRequest(b.in, card, art, scale))
	}
	if err := b.renderScales(planned, renderScale); err != nil {
		return nil, fmt.Errorf("card %q: %w", card.ID, err)
	}

	b.cache.set(cacheName, key)
	b.rendered.Add(int64(len(b.opts.Scales)))

	return planned, nil
}

func (b *build) buildBack(ctx context.Context) ([]model.Image, error) {
	planned := b.plannedImages("", backBaseName)
	key := keyOf(version, backBaseName, b.renderer.Version(), b.in.themeHash,
		joinInts(b.opts.Scales), fmt.Sprint(b.opts.Formats))

	if b.upToDate(backBaseName, key, planned) {
		b.skipped.Add(int64(len(b.opts.Scales)))

		return planned, nil
	}

	renderScale := func(scale int) ([]byte, error) {
		return b.renderer.RenderBack(ctx, render.BackRequest{Theme: b.in.theme, Scale: scale})
	}
	if err := b.renderScales(planned, renderScale); err != nil {
		return nil, fmt.Errorf("back: %w", err)
	}

	b.cache.set(backBaseName, key)
	b.rendered.Add(int64(len(b.opts.Scales)))

	return planned, nil
}

func (b *build) upToDate(cacheName, key string, planned []model.Image) bool {
	if b.opts.Force || !b.cache.matches(cacheName, key) {
		return false
	}

	paths := make([]string, len(planned))
	for i, img := range planned {
		paths[i] = img.Path
	}

	return allExist(b.outDir, paths)
}

// renderScales renders each scale once and writes it in every format.
func (b *build) renderScales(planned []model.Image, renderScale func(scale int) ([]byte, error)) error {
	byScale := map[int][]model.Image{}
	for _, img := range planned {
		byScale[img.Scale] = append(byScale[img.Scale], img)
	}

	for _, scale := range b.opts.Scales {
		images := byScale[scale]

		pngData, err := renderScale(scale)
		if err != nil {
			return fmt.Errorf("render at %dx: %w", scale, err)
		}
		if err := checkPNGSize(pngData, images[0].Width, images[0].Height); err != nil {
			return fmt.Errorf("render at %dx: %w", scale, err)
		}

		encoded, err := encodeAll(pngData, b.opts.Formats)
		if err != nil {
			return err
		}
		for _, img := range images {
			target := filepath.Join(b.outDir, filepath.FromSlash(img.Path))
			if err := writeFileAtomic(target, encoded[img.Format]); err != nil {
				return fmt.Errorf("write %s: %w", img.Path, err)
			}
		}
	}

	return nil
}

func renderCardRequest(in inputs, card i18n.ResolvedCard, art []byte, scale int) render.CardRequest {
	return render.CardRequest{Theme: in.theme, Card: card, Art: art, Scale: scale}
}
