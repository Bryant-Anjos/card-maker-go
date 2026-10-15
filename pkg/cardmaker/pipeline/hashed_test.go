package pipeline_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/pipeline"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
)

var hashedName = regexp.MustCompile(`^(.+?)(@\dx)?\.([0-9a-f]{8})\.(png|webp)$`)

func hashedOptions(ws workspace) pipeline.Options {
	opts := ws.options()
	opts.HashedNames = true

	return opts
}

func allImages(m model.Manifest) []model.Image {
	images := append([]model.Image{}, m.Back...)
	for _, card := range m.Cards {
		images = append(images, card.Images...)
	}

	return images
}

func listImages(t *testing.T, dir string) []string {
	t.Helper()

	var names []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}

	return names
}

func TestHashedNamesMatchContent(t *testing.T) {
	ws := newWorkspace(t)

	result := mustRun(t, hashedOptions(ws), render.NewFake())

	images := allImages(result.Manifest)
	if len(images) != 4+5*4 {
		t.Fatalf("got %d images, want 24", len(images))
	}
	seen := map[string]bool{}
	for _, img := range images {
		data, err := os.ReadFile(filepath.Join(ws.localeDir(), filepath.FromSlash(img.Path)))
		if err != nil {
			t.Fatalf("listed file missing: %v", err)
		}
		sum := sha256.Sum256(data)
		want := hex.EncodeToString(sum[:])[:8]
		m := hashedName.FindStringSubmatch(filepath.Base(img.Path))
		if m == nil || m[3] != want {
			t.Errorf("%s: name does not carry the content hash %s", img.Path, want)
		}
		seen[img.Path] = true
	}
	if len(seen) != len(images) {
		t.Error("image paths are not unique")
	}
	if got := result.Manifest.Back[0].Path; !strings.HasPrefix(got, "back.") || strings.Contains(got, "/") {
		t.Errorf("back stays in the root, got %q", got)
	}
	if got := result.Manifest.Cards[0].Images[0].Path; !strings.HasPrefix(got, "cards/vulpin.") {
		t.Errorf("cards stay in cards/, got %q", got)
	}
	if got := result.Manifest.Cards[0].Images[2].Path; !strings.Contains(got, "@2x.") {
		t.Errorf("scale suffix kept before the hash, got %q", got)
	}
}

func TestHashedNamesManifestOnDiskMatches(t *testing.T) {
	ws := newWorkspace(t)
	result := mustRun(t, hashedOptions(ws), render.NewFake())

	file, err := os.Open(filepath.Join(ws.localeDir(), "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	onDisk, err := model.ParseManifest(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(allImages(onDisk)) != len(allImages(result.Manifest)) {
		t.Fatal("manifest on disk differs from the result")
	}
	if onDisk.Cards[0].Name != "Vulpin" || onDisk.Cards[0].Images[0].Width != 75 {
		t.Errorf("other fields changed: %+v", onDisk.Cards[0])
	}
}

func TestHashedNamesAreStableAndTrackContent(t *testing.T) {
	ws := newWorkspace(t)
	first := mustRun(t, hashedOptions(ws), render.NewFake())

	changeLunaris(t, ws)
	second := mustRun(t, hashedOptions(ws), render.NewFake())

	for i, card := range first.Manifest.Cards {
		same := card.Images[0].Path == second.Manifest.Cards[i].Images[0].Path
		if card.ID == "lunaris" && same {
			t.Error("lunaris changed but its file name did not")
		}
		if card.ID != "lunaris" && !same {
			t.Errorf("%s did not change but its file name did", card.ID)
		}
	}
	if first.Manifest.Back[0].Path != second.Manifest.Back[0].Path {
		t.Error("back name changed without a change")
	}

	// A fresh directory with the same inputs produces the same names.
	other := newWorkspace(t)
	changeLunaris(t, other)
	third := mustRun(t, hashedOptions(other), render.NewFake())
	if third.Manifest.Cards[2].Images[0].Path != second.Manifest.Cards[2].Images[0].Path {
		t.Error("same content produced different names in another directory")
	}
}

func TestWithoutHashedNamesNothingChanges(t *testing.T) {
	ws := newWorkspace(t)
	result := mustRun(t, ws.options(), render.NewFake())

	if got := result.Manifest.Back[0].Path; got != "back.png" {
		t.Errorf("back path = %q", got)
	}
	if got := result.Manifest.Cards[0].Images[3].Path; got != "cards/vulpin@2x.webp" {
		t.Errorf("card path = %q", got)
	}
}

func TestHashedNamesSecondRunRendersNothing(t *testing.T) {
	ws := newWorkspace(t)
	fake := render.NewFake()
	first := mustRun(t, hashedOptions(ws), fake)
	before := fake.CardCalls() + fake.BackCalls()

	second := mustRun(t, hashedOptions(ws), fake)

	if got := fake.CardCalls() + fake.BackCalls(); got != before {
		t.Errorf("renderer called %d more times", got-before)
	}
	if second.Rendered != 0 || second.Skipped != 12 {
		t.Errorf("result = %+v, want 0 rendered, 12 skipped", second)
	}
	for i, img := range allImages(first.Manifest) {
		if allImages(second.Manifest)[i].Path != img.Path {
			t.Errorf("name changed on a cached run: %s", img.Path)
		}
	}
}

func TestHashedNamesRerenderWhenAFileIsMissing(t *testing.T) {
	ws := newWorkspace(t)
	fake := render.NewFake()
	first := mustRun(t, hashedOptions(ws), fake)
	missing := first.Manifest.Cards[1].Images[0].Path
	if err := os.Remove(filepath.Join(ws.localeDir(), filepath.FromSlash(missing))); err != nil {
		t.Fatal(err)
	}

	second := mustRun(t, hashedOptions(ws), fake)

	if second.Rendered != 2 {
		t.Errorf("rendered %d, want 2 (one card)", second.Rendered)
	}
	if _, err := os.Stat(filepath.Join(ws.localeDir(), filepath.FromSlash(missing))); err != nil {
		t.Errorf("file not restored: %v", err)
	}
}

func TestHashedNamesAfterUnhashedGeneration(t *testing.T) {
	ws := newWorkspace(t)
	fake := render.NewFake()
	mustRun(t, ws.options(), fake)

	result := mustRun(t, hashedOptions(ws), fake)

	for _, img := range allImages(result.Manifest) {
		if !hashedName.MatchString(filepath.Base(img.Path)) {
			t.Errorf("not hashed: %s", img.Path)
		}
		if _, err := os.Stat(filepath.Join(ws.localeDir(), filepath.FromSlash(img.Path))); err != nil {
			t.Errorf("missing: %v", err)
		}
	}
	for _, name := range listImages(t, ws.localeDir()) {
		if name == "back.png" || name == "back@2x.webp" {
			t.Errorf("old unhashed file %s was kept", name)
		}
	}
	for _, name := range listImages(t, filepath.Join(ws.localeDir(), "cards")) {
		if !hashedName.MatchString(name) {
			t.Errorf("old unhashed card file %s was kept", name)
		}
	}
}

func TestHashedNamesBackToUnhashed(t *testing.T) {
	ws := newWorkspace(t)
	fake := render.NewFake()
	mustRun(t, hashedOptions(ws), fake)

	result := mustRun(t, ws.options(), fake)

	if got := result.Manifest.Back[0].Path; got != "back.png" {
		t.Errorf("back path = %q", got)
	}
	if _, err := os.Stat(filepath.Join(ws.localeDir(), "cards", "vulpin.png")); err != nil {
		t.Errorf("unhashed file not written: %v", err)
	}
}

func TestHashedNamesCleanupKeepsOneGeneration(t *testing.T) {
	ws := newWorkspace(t)
	fake := render.NewFake()
	first := mustRun(t, hashedOptions(ws), fake)
	oldPath := first.Manifest.Cards[2].Images[0].Path

	changeLunaris(t, ws)
	second := mustRun(t, hashedOptions(ws), fake)

	if _, err := os.Stat(filepath.Join(ws.localeDir(), filepath.FromSlash(oldPath))); !os.IsNotExist(err) {
		t.Errorf("old generation file %s should be gone (err=%v)", oldPath, err)
	}
	want := map[string]bool{}
	for _, img := range allImages(second.Manifest) {
		want[img.Path] = true
	}
	var found int
	for _, name := range listImages(t, filepath.Join(ws.localeDir(), "cards")) {
		if !want["cards/"+name] {
			t.Errorf("orphan left in cards/: %s", name)
		}
		found++
	}
	for _, name := range listImages(t, ws.localeDir()) {
		if strings.HasPrefix(name, "back") && !want[name] {
			t.Errorf("orphan left in root: %s", name)
		}
		if strings.HasPrefix(name, "back") {
			found++
		}
	}
	if found != len(want) {
		t.Errorf("%d image files on disk, manifest lists %d", found, len(want))
	}
}

func TestHashedNamesCleanupPreservesStrangers(t *testing.T) {
	ws := newWorkspace(t)
	fake := render.NewFake()
	mustRun(t, hashedOptions(ws), fake)

	keep := []string{
		filepath.Join(ws.localeDir(), "notes.txt"),
		filepath.Join(ws.localeDir(), "cards", "readme.txt"),
		filepath.Join(ws.localeDir(), "cards", ".keep"),
		filepath.Join(ws.outDir, "outside.webp"),
		filepath.Join(ws.outDir, "outside.txt"),
		filepath.Join(ws.outDir, "pt-BR", "cards", "sub", "nested.webp"),
	}
	for _, p := range keep {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Another locale next to this one is never touched.
	otherLocale := filepath.Join(ws.outDir, "en-US", "cards", "old.webp")
	if err := os.MkdirAll(filepath.Dir(otherLocale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otherLocale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	keep = append(keep, otherLocale,
		filepath.Join(ws.localeDir(), "manifest.json"), filepath.Join(ws.localeDir(), ".cache.json"))

	changeLunaris(t, ws)
	mustRun(t, hashedOptions(ws), fake)

	for _, p := range keep {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s should have been preserved: %v", p, err)
		}
	}
}

func TestUnhashedBuildNeverDeletes(t *testing.T) {
	ws := newWorkspace(t)
	fake := render.NewFake()
	mustRun(t, ws.options(), fake)
	stray := filepath.Join(ws.localeDir(), "cards", "stray.webp")
	if err := os.WriteFile(stray, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	mustRun(t, ws.options(), fake)

	if _, err := os.Stat(stray); err != nil {
		t.Errorf("a build without -hashed-names must not clean: %v", err)
	}
}

// changeLunaris changes the rendered content of one card. The fake renderer
// paints the rarity color and ignores the art, so the rarity is what changes.
func changeLunaris(t *testing.T, ws workspace) {
	t.Helper()

	replaceInFile(t, ws.cardsFile, `"rarity": "rare"`, `"rarity": "epic"`)
}
