package docscheck

import (
	"reflect"
	"regexp"
	"sort"
	"testing"
)

var (
	flagCall    = regexp.MustCompile(`fs\.(?:String|Bool|Int|Int64|Duration)\(\s*"([a-z][a-z-]*)"`)
	flagVarCall = regexp.MustCompile(`fs\.\w+Var\([^,]+,\s*"([a-z][a-z-]*)"`)
	envName     = regexp.MustCompile(`"(CARDMAKER_[A-Z_]+)"`)
	routeCall   = regexp.MustCompile(`"(GET|POST|PUT|PATCH|DELETE) (/[^"]*)"`)
	commandCase = regexp.MustCompile(`(?m)^\tcase "([a-z]+)":`)
	makeTarget  = regexp.MustCompile(`(?m)^([a-z][a-z-]*):`)
)

func unique(values []string) []string {
	set := map[string]bool{}
	for _, v := range values {
		set[v] = true
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)

	return out
}

func submatches(re *regexp.Regexp, src string, group int) []string {
	var out []string
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		out = append(out, m[group])
	}

	return out
}

func flagsIn(src string) []string {
	return unique(append(submatches(flagCall, src, 1), submatches(flagVarCall, src, 1)...))
}

func TestExtractFlags(t *testing.T) {
	src := `
	addr := fs.String("addr", "127.0.0.1:8080", "address")
	force := fs.Bool("force", false, "x")
	n := fs.Int("max-concurrent", 4, "x")
	d := fs.Duration("queue-timeout", 0, "x")
	fs.StringVar(&c.cards, "cards", "", "x")
	other := fmt.Sprintf("not-a-flag")
	`

	want := []string{"addr", "cards", "force", "max-concurrent", "queue-timeout"}

	if got := flagsIn(src); !reflect.DeepEqual(got, want) {
		t.Errorf("flags = %v, want %v", got, want)
	}
}

func TestExtractEnvRoutesCommandsAndTargets(t *testing.T) {
	if got := unique(submatches(envName, `const a = "CARDMAKER_CHROME"; b := "CARDMAKER_API_TOKEN"; c := "OTHER_VAR"`, 1)); !reflect.DeepEqual(got, []string{"CARDMAKER_API_TOKEN", "CARDMAKER_CHROME"}) {
		t.Errorf("env = %v", got)
	}

	routes := submatches(routeCall, `mux.Handle("GET /v1/themes", h); mux.HandleFunc("POST /v1/render", f)`, 0)
	if want := []string{`"GET /v1/themes"`, `"POST /v1/render"`}; !reflect.DeepEqual(routes, want) {
		t.Errorf("routes = %v, want %v", routes, want)
	}

	if got := submatches(commandCase, "switch args[0] {\n\tcase \"version\":\n\tcase \"build\":\n\t\tcase \"nested\":\n", 1); !reflect.DeepEqual(got, []string{"version", "build"}) {
		t.Errorf("commands = %v", got)
	}

	if got := submatches(makeTarget, ".PHONY: a b\nbuild:\n\tgo build\ntest-all:\n\tx\n\tnot-a-target:\n", 1); !reflect.DeepEqual(got, []string{"build", "test-all"}) {
		t.Errorf("targets = %v", got)
	}
}
