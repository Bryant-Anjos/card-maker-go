package server

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/theme"
)

// LoadThemes loads every theme found in the subdirectories of dir, keyed by
// directory name. A subdirectory without theme.json is ignored; a theme that
// fails to load is an error, so a broken theme never goes unnoticed.
func LoadThemes(dir string) (map[string]theme.Theme, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("themes directory: %w", err)
	}

	themes := make(map[string]theme.Theme)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		themeDir := filepath.Join(dir, entry.Name())
		if _, err := os.Stat(filepath.Join(themeDir, theme.DefinitionFile)); err != nil {
			continue
		}

		loaded, err := theme.LoadDir(themeDir)
		if err != nil {
			return nil, fmt.Errorf("theme %q: %w", entry.Name(), err)
		}
		themes[entry.Name()] = loaded
	}
	if len(themes) == 0 {
		return nil, fmt.Errorf("no themes found in %q", dir)
	}

	return themes, nil
}

type themeSummary struct {
	Name     string     `json:"name"`
	Size     model.Size `json:"size"`
	Rarities []string   `json:"rarities"`
	Elements []string   `json:"elements"`
	Variants []string   `json:"variants"`
	HasBack  bool       `json:"hasBack"`
}

func summarize(name string, t theme.Theme) themeSummary {
	summary := themeSummary{
		Name:     name,
		Size:     t.Size,
		Rarities: make([]string, 0, len(t.Rarities)),
		Elements: make([]string, 0, len(t.Elements)),
		Variants: make([]string, 0, len(t.Variants)),
		HasBack:  t.Back != "",
	}
	for _, r := range t.Rarities {
		summary.Rarities = append(summary.Rarities, r.ID)
	}
	for _, e := range t.Elements {
		summary.Elements = append(summary.Elements, e.ID)
	}
	for _, v := range t.Variants {
		summary.Variants = append(summary.Variants, v.ID)
	}

	return summary
}

func (s *Server) listThemes() []themeSummary {
	names := make([]string, 0, len(s.cfg.Themes))
	for name := range s.cfg.Themes {
		names = append(names, name)
	}
	sort.Strings(names)

	summaries := make([]themeSummary, len(names))
	for i, name := range names {
		summaries[i] = summarize(name, s.cfg.Themes[name])
	}

	return summaries
}

func (s *Server) themeNamed(name string) (theme.Theme, *apiError) {
	if name == "" {
		return theme.Theme{}, badRequest("query parameter 'theme' is required")
	}
	t, ok := s.cfg.Themes[name]
	if !ok {
		return theme.Theme{}, notFound(fmt.Sprintf("unknown theme %q", name))
	}

	return t, nil
}
