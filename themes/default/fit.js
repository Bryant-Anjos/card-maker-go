// Shrinks every element marked with data-fit until its text fits.
//
//   <h1 class="name" data-fit data-fit-min="34">...</h1>
//   <p class="description" data-fit="lines" data-fit-min="26">...</p>
//
// The size starts at the element's CSS font-size and goes down, one pixel at a
// time, to data-fit-min.
//
// - Plain data-fit: the text must fit on ONE line (not wider than the box).
//   If it still does not fit at the minimum, the element's own CSS (overflow
//   hidden + text-overflow: ellipsis) cuts it.
// - data-fit="lines": the text may use as many lines as the box's height holds
//   (the element needs a max-height; smaller text fits in more lines). If it
//   still does not fit at the minimum, the last line that fits ends with an
//   ellipsis (-webkit-line-clamp is set to the lines that fit).
//
// It runs after the fonts are loaded, because the size of the text depends on
// them. The renderer waits for window.cardReady before taking the screenshot.
(() => {
  // Rounding and the font's own metrics make scrollHeight a few pixels taller
  // than the lines really are, even when the text fits; one more line is always
  // far taller than this.
  const HEIGHT_SLACK = 3;

  // The height the text may take: the element's max-height (its box is only as
  // tall as the text, so its own height says nothing about what fits).
  const heightLimit = (el) => {
    const max = parseFloat(getComputedStyle(el).maxHeight);
    return Number.isFinite(max) ? max : el.clientHeight;
  };

  // scrollWidth and clientWidth are whole pixels, so a text wider than its box by
  // a fraction of a pixel looked like it fit; the browser then cut its last
  // letters with an ellipsis ("Serpente Mari..."), at random for names that land on
  // that boundary. The real (fractional) width of the text is measured instead.
  const textWidth = (el) => {
    const range = document.createRange();
    range.selectNodeContents(el);
    return range.getBoundingClientRect().width;
  };
  const overflowsWidth = (el) => textWidth(el) > el.getBoundingClientRect().width - 1;
  const overflowsHeight = (el) => el.scrollHeight > heightLimit(el) + HEIGHT_SLACK;

  const fitText = (el) => {
    const byLines = el.dataset.fit === 'lines';
    const overflows = byLines ? overflowsHeight : overflowsWidth;
    const max = Math.floor(parseFloat(getComputedStyle(el).fontSize));
    const min = Math.ceil(parseFloat(el.dataset.fitMin) || max);
    if (min >= max || !overflows(el)) return;

    // Binary search for the largest integer size, from min up to max - 1, that
    // fits; the minimum is the fallback when nothing does.
    let low = min;
    let high = max - 1;
    let best = min;
    while (low <= high) {
      const mid = (low + high) >> 1;
      el.style.fontSize = `${mid}px`;
      if (overflows(el)) {
        high = mid - 1;
      } else {
        best = mid;
        low = mid + 1;
      }
    }
    el.style.fontSize = `${best}px`;
    if (byLines && overflows(el)) clampToLines(el);
  };

  // At the minimum size the text is still too tall: keep the lines that fit and
  // let the browser draw the ellipsis on the last one.
  const clampToLines = (el) => {
    const lineHeight = parseFloat(getComputedStyle(el).lineHeight);
    const lines = Math.max(1, Math.floor((heightLimit(el) + HEIGHT_SLACK) / lineHeight));
    el.style.webkitLineClamp = String(lines);
  };

  const afterLoad = new Promise((resolve) => {
    if (document.readyState === 'complete') resolve();
    else window.addEventListener('load', resolve, { once: true });
  });

  window.cardReady = afterLoad
    .then(() => document.fonts.ready)
    .then(() => document.querySelectorAll('[data-fit]').forEach(fitText));
})();
