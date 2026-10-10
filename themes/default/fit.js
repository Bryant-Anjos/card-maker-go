// Shrinks every element marked with data-fit until its text fits on one line.
//
//   <h1 class="name" data-fit data-fit-min="34">...</h1>
//
// The size starts at the element's CSS font-size and goes down, one pixel at a
// time, to data-fit-min. If the text still does not fit at the minimum, the
// element's own CSS (overflow hidden + text-overflow: ellipsis) cuts it.
//
// It runs after the fonts are loaded, because the width of the text depends on
// them. The renderer waits for window.cardReady before taking the screenshot.
(() => {
  const overflows = (el) => el.scrollWidth > el.clientWidth;

  const fitText = (el) => {
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
  };

  const afterLoad = new Promise((resolve) => {
    if (document.readyState === 'complete') resolve();
    else window.addEventListener('load', resolve, { once: true });
  });

  window.cardReady = afterLoad
    .then(() => document.fonts.ready)
    .then(() => document.querySelectorAll('[data-fit]').forEach(fitText));
})();
