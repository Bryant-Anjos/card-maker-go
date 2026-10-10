// Package chrome renders cards by loading the theme's HTML/CSS templates in a
// headless Chromium-based browser and taking a screenshot.
package chrome

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
)

// rendererVersion is bumped whenever this package changes its output.
const rendererVersion = "1"

const defaultTimeout = 60 * time.Second

// waitForAssets resolves once web fonts are ready, the theme's own asynchronous
// work is done and every image decoded, so screenshots never catch a
// half-loaded card.
//
// A theme that needs to run code after the fonts load (for example to fit text)
// assigns a promise to window.cardReady; the renderer waits for it, and a
// rejection fails the render. Themes that define nothing are unaffected.
const waitForAssets = `Promise.all([document.fonts.ready, window.cardReady])
	.then(() => Promise.all(Array.from(document.images, (img) => img.decode().catch(() => null))))
	.then(() => new Promise((resolve) => requestAnimationFrame(() => resolve(true))))`

// setTransparentBackground is Emulation.setDefaultBackgroundColorOverride with
// parameters that always send the alpha channel: cdp.RGBA drops a zero alpha
// (omitzero) and the browser then defaults to opaque.
var setTransparentBackground = cdp.Command[transparentBackgroundParams, cdp.Empty]{
	Method: emulation.CommandSetDefaultBackgroundColorOverride,
}

type transparentBackgroundParams struct {
	Color struct {
		R int64   `json:"r"`
		G int64   `json:"g"`
		B int64   `json:"b"`
		A float64 `json:"a"`
	} `json:"color"`
}

// Options configures a Renderer.
type Options struct {
	BrowserPath string        // browser executable; see FindBrowser for the default lookup
	Timeout     time.Duration // limit per rendered image, default 60s

	// NoSandbox launches the browser with --no-sandbox. Only enable it inside
	// a container (or similar) that already isolates the process: containers
	// usually forbid the user namespaces the browser sandbox needs.
	NoSandbox bool
}

// Renderer implements render.Renderer with a headless browser. Create it with
// New and always Close it. It is safe for concurrent use: each image renders
// in its own tab.
type Renderer struct {
	server        *server
	browserCtx    context.Context
	cancelBrowser context.CancelFunc
	cancelAlloc   context.CancelFunc
	timeout       time.Duration
	version       string

	closeOnce sync.Once
	closeErr  error
}

// New starts the browser and the local page server.
func New(opts Options) (*Renderer, error) {
	path, err := FindBrowser(opts.BrowserPath)
	if err != nil {
		return nil, err
	}

	srv, err := newServer()
	if err != nil {
		return nil, err
	}

	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(path),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("font-render-hinting", "none"),
		chromedp.Flag("force-color-profile", "srgb"),
	)
	if opts.NoSandbox {
		allocOpts = append(allocOpts, chromedp.NoSandbox)
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)

	r := &Renderer{
		server:        srv,
		browserCtx:    browserCtx,
		cancelBrowser: cancelBrowser,
		cancelAlloc:   cancelAlloc,
		timeout:       opts.Timeout,
	}
	if r.timeout <= 0 {
		r.timeout = defaultTimeout
	}

	info, err := chromedp.CallBrowser(browserCtx, browser.GetVersion, cdp.Empty{})
	if err != nil {
		_ = r.Close()
		return nil, fmt.Errorf("start browser %q: %w", path, err)
	}
	r.version = "chrome-" + rendererVersion + "/" + info.Product

	return r, nil
}

// Version implements render.Renderer. It includes the browser version, so a
// browser upgrade invalidates the pipeline cache.
func (r *Renderer) Version() string { return r.version }

// Close stops the browser and the page server.
func (r *Renderer) Close() error {
	r.closeOnce.Do(func() {
		r.cancelBrowser()
		r.cancelAlloc()
		r.closeErr = r.server.close()
	})

	return r.closeErr
}

// RenderCard implements render.Renderer.
func (r *Renderer) RenderCard(ctx context.Context, req render.CardRequest) ([]byte, error) {
	html, err := cardPage(req.Theme, req.Card)
	if err != nil {
		return nil, err
	}

	return r.capture(ctx, pageContent{html: html, art: req.Art, fsys: req.Theme.FS}, req.Theme.Size, req.Scale)
}

// RenderBack implements render.Renderer.
func (r *Renderer) RenderBack(ctx context.Context, req render.BackRequest) ([]byte, error) {
	html, err := backPage(req.Theme)
	if err != nil {
		return nil, err
	}

	return r.capture(ctx, pageContent{html: html, fsys: req.Theme.FS}, req.Theme.Size, req.Scale)
}

func (r *Renderer) capture(ctx context.Context, p pageContent, size model.Size, scale int) ([]byte, error) {
	if scale < 1 {
		return nil, fmt.Errorf("scale must be at least 1, got %d", scale)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.browserCtx.Err(); err != nil {
		return nil, errors.New("renderer is closed")
	}

	url, release := r.server.register(p)
	defer release()

	tabCtx, cancelTab := chromedp.NewContext(r.browserCtx)
	defer cancelTab()
	stop := context.AfterFunc(ctx, cancelTab)
	defer stop()
	tabCtx, cancelTimeout := context.WithTimeout(tabCtx, r.timeout)
	defer cancelTimeout()

	screenshot, err := r.screenshot(tabCtx, url, size, scale)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, fmt.Errorf("render in browser: %w", err)
	}

	return screenshot, nil
}

// screenshot loads url in the tab of ctx at the given size and device scale
// and captures the viewport as PNG.
func (r *Renderer) screenshot(ctx context.Context, url string, size model.Size, scale int) ([]byte, error) {
	if err := r.loadPage(ctx, url, size, scale); err != nil {
		return nil, err
	}

	return chromedp.Run(ctx, chromedp.CaptureScreenshot())
}

// loadPage opens url in the tab of ctx at the given size and device scale and
// waits until the page is ready to be captured (see waitForAssets).
func (r *Renderer) loadPage(ctx context.Context, url string, size model.Size, scale int) error {
	err := chromedp.Do(ctx,
		chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
			_, err := cdp.Call(ctx, t, emulation.SetDeviceMetricsOverride, emulation.SetDeviceMetricsOverrideParams{
				Width:             int64(size.Width),
				Height:            int64(size.Height),
				DeviceScaleFactor: float64(scale),
			})
			return err
		}),
		chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
			_, err := cdp.Call(ctx, t, setTransparentBackground, transparentBackgroundParams{})
			return err
		}),
		chromedp.Navigate(url),
	)
	if err != nil {
		return err
	}

	if _, err := chromedp.Run(ctx, chromedp.Evaluate[bool](waitForAssets, chromedp.EvalAwaitPromise)); err != nil {
		return fmt.Errorf("wait for fonts, images and the theme: %w", err)
	}

	return nil
}
