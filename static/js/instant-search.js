// Keep search controls mounted so typing retains focus and cursor position.
// The server renders and authorizes every result, including pagination.
document.querySelectorAll('[data-instant-region]').forEach((region) => {
  const form = region.querySelector('form[data-instant-filter]');
  const results = region.querySelector('[data-instant-results]');
  const feedback = region.querySelector('[data-instant-status]');
  if (!form || !results || !feedback || form.method.toLowerCase() !== 'get') return;

  let timer;
  let request;
  let revision = 0;
  let composing = false;
  let retryURL;
  const action = new URL(form.action, location.href);

  const message = (text, error = false) => {
    feedback.textContent = text;
    feedback.toggleAttribute('data-error', error);
    if (error) {
      const retry = document.createElement('button');
      retry.type = 'button';
      retry.className = 'instant-retry';
      retry.textContent = 'Coba lagi';
      retry.addEventListener('click', () => load(retryURL ?? queryURL(), 'replace'));
      feedback.append(retry);
    }
  };
  const cancel = () => {
    clearTimeout(timer);
    revision += 1;
    request?.abort();
    results.setAttribute('aria-busy', 'false');
  };
  const queryURL = () => {
    const url = new URL(action);
    url.search = new URLSearchParams(new FormData(form)).toString();
    url.searchParams.delete('limit'); // Every table uses ten rows per page.
    url.searchParams.set('page', '1');
    return url;
  };
  const validInputs = () => [...form.querySelectorAll('input')].every((input) => input.checkValidity());

  const syncControls = (nextForm, restore) => {
    for (const field of form.elements) {
      if (!field.name) continue;
      const next = nextForm.elements.namedItem(field.name);
      if (!next) continue;
      if (field instanceof HTMLSelectElement) {
        field.innerHTML = next.innerHTML;
        field.value = next.value;
        field.disabled = next.disabled;
      } else if (restore && field instanceof HTMLInputElement) {
        field.value = next.value;
      }
    }
  };

  const load = async (url, historyMode = 'replace', restore = false) => {
    cancel();
    retryURL = url;
    const current = revision;
    request = new AbortController();
    if (!restore && !validInputs()) {
      message('Lengkapi isian filter untuk menampilkan hasil.');
      return;
    }
    results.setAttribute('aria-busy', 'true');
    message('Memuat hasil…');
    try {
      const response = await fetch(url, {
        signal: request.signal,
        credentials: 'same-origin',
        cache: 'no-store',
        headers: { Accept: 'text/html' },
      });
      if (!response.ok) throw new Error('Hasil belum dapat dimuat.');
      const html = await response.text();
      if (current !== revision) return;
      const page = new DOMParser().parseFromString(html, 'text/html');
      const nextRegion = page.getElementById(region.id);
      const nextResults = nextRegion?.querySelector('[data-instant-results]');
      const nextForm = nextRegion?.querySelector('form[data-instant-filter]');
      if (!nextResults || !nextForm) throw new Error('Halaman hasil tidak tersedia.');

      // Follow the server's canonical page after filtered rows disappear.
      if (response.redirected) url = new URL(response.url);
      results.innerHTML = nextResults.innerHTML;
      syncControls(nextForm, restore);
      const summary = region.querySelector('[data-instant-summary]');
      const nextSummary = nextRegion.querySelector('[data-instant-summary]');
      if (summary && nextSummary) summary.textContent = nextSummary.textContent;
      // Class/teacher creation dialogs follow the selected academic year.
      page.querySelectorAll('[data-instant-sync][id]').forEach((next) => {
        const existing = document.getElementById(next.id);
        if (existing && !existing.open) existing.replaceWith(document.importNode(next, true));
      });
      if (historyMode !== 'none') {
        const state = { ...history.state, instantRegion: region.id };
        if (historyMode === 'push' && url.href !== location.href) history.pushState(state, '', url);
        else history.replaceState(state, '', url);
      }
      message(summary ? summary.textContent : 'Hasil diperbarui.');
    } catch (error) {
      if (error.name !== 'AbortError' && current === revision) {
        message('Hasil gagal dimuat. Periksa koneksi atau sesi login Anda.', true);
      }
    } finally {
      if (current === revision) results.setAttribute('aria-busy', 'false');
    }
  };

  const schedule = () => {
    cancel();
    message('Mencari…');
    timer = setTimeout(() => load(queryURL()), 300);
  };
  form.addEventListener('input', (event) => {
    if (event.target instanceof HTMLInputElement && !composing && !event.isComposing) schedule();
  });
  form.addEventListener('compositionstart', () => { composing = true; cancel(); });
  form.addEventListener('compositionend', () => { composing = false; schedule(); });
  form.addEventListener('change', (event) => {
    if (!(event.target instanceof HTMLSelectElement)) return;
    const dependentName = event.target.dataset.resetFilter;
    const dependent = dependentName && form.elements.namedItem(dependentName);
    if (dependent instanceof HTMLSelectElement) {
      dependent.value = '';
      // Discard stale options immediately while the server loads this year.
      dependent.replaceChildren(dependent.options[0].cloneNode(true));
      dependent.disabled = true;
    }
    load(queryURL(), 'push');
  });
  form.addEventListener('submit', (event) => {
    event.preventDefault();
    load(queryURL());
  });
  region.addEventListener('click', (event) => {
    const link = event.target.closest('a[href]');
    if (!link || !results.contains(link) || event.defaultPrevented || event.button !== 0 ||
        event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || link.target || link.hasAttribute('download')) return;
    const url = new URL(link.href);
    if (url.origin !== action.origin || url.pathname !== action.pathname) return;
    event.preventDefault();
    load(url, 'push');
  });
  window.addEventListener('popstate', () => {
    const url = new URL(location.href);
    if (url.pathname === action.pathname) load(url, 'none', true);
  });
});
