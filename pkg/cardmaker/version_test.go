package cardmaker_test

import (
	"regexp"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker"
)

func TestVersionIsSemver(t *testing.T) {
	semver := regexp.MustCompile(`^\d+\.\d+\.\d+$`)

	if !semver.MatchString(cardmaker.Version) {
		t.Fatalf("Version %q is not semver", cardmaker.Version)
	}
}
