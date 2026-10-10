package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/pipeline"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
)

// Exit codes.
const (
	exitOK      = 0
	exitFailure = 1 // the command ran and failed
	exitUsage   = 2 // the command line itself was wrong
)

const usage = `usage: cardmaker <command> [flags]

commands:
  version    print the version
  validate   check cards data, theme and artwork without rendering
  build      render every card and write the manifest

run "cardmaker <command> -h" for the flags of a command.`

// run executes the CLI and returns the process exit code. It is separate from
// main so it can be tested.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return exitUsage
	}

	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, cardmaker.Version)
		return exitOK
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "build":
		return runBuild(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s\n", args[0], usage)
		return exitUsage
	}
}

// commonFlags are shared by validate and build.
type commonFlags struct {
	cards, theme, locale, fallback string
}

func (c *commonFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&c.cards, "cards", "", "cards data file (required)")
	fs.StringVar(&c.theme, "theme", "", "theme directory (required)")
	fs.StringVar(&c.locale, "locale", "pt-BR", "output locale")
	fs.StringVar(&c.fallback, "fallback", "", "fallback locale (default: same as -locale)")
}

func (c commonFlags) options() pipeline.Options {
	return pipeline.Options{CardsFile: c.cards, ThemeDir: c.theme, Locale: c.locale, Fallback: c.fallback}
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var common commonFlags
	common.register(fs)

	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	count, err := pipeline.Validate(common.options())
	if err != nil {
		fmt.Fprintln(stderr, "validation failed:", err)
		return exitFailure
	}

	fmt.Fprintf(stdout, "OK: %d cards are valid for theme %q and locale %s\n", count, common.theme, common.locale)

	return exitOK
}

func runBuild(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var common commonFlags
	common.register(fs)
	out := fs.String("out", "", "output directory (required); files go to <out>/<locale>/")
	scales := fs.String("scales", "1,2", "comma-separated image scales")
	formats := fs.String("formats", "png", "comma-separated image formats: png, webp")
	concurrency := fs.Int("concurrency", 0, "cards rendered in parallel (default 4)")
	force := fs.Bool("force", false, "ignore the cache and render everything")
	rendererName := fs.String("renderer", "chrome", "renderer: chrome or fake")

	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	parsedScales, err := parseScales(*scales)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}

	opts := common.options()
	opts.OutDir = *out
	opts.Scales = parsedScales
	opts.Formats = splitList(*formats)
	opts.Concurrency = *concurrency
	opts.Force = *force

	renderer, err := newRenderer(*rendererName)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailure
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	result, err := pipeline.Run(ctx, opts, renderer)
	if err != nil {
		fmt.Fprintln(stderr, "build failed:", err)
		return exitFailure
	}

	fmt.Fprintf(stdout, "Built %d cards (theme %q, locale %s) in %s: rendered %d, skipped %d\n",
		len(result.Manifest.Cards), result.Manifest.Theme, result.Manifest.Locale,
		result.OutDir, result.Rendered, result.Skipped)

	return exitOK
}

func newRenderer(name string) (render.Renderer, error) {
	switch name {
	case "fake":
		return render.NewFake(), nil
	case "chrome":
		return nil, errors.New("the chrome renderer is not implemented yet; use -renderer fake to exercise the pipeline")
	default:
		return nil, fmt.Errorf("unknown renderer %q, use chrome or fake", name)
	}
}

func parseScales(value string) ([]int, error) {
	var scales []int
	for _, item := range splitList(value) {
		scale, err := strconv.Atoi(item)
		if err != nil {
			return nil, fmt.Errorf("invalid scale %q: must be a whole number", item)
		}
		scales = append(scales, scale)
	}

	return scales, nil
}

// splitList splits a comma-separated flag value, trimming spaces and dropping
// empty items.
func splitList(value string) []string {
	var items []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}

	return items
}
