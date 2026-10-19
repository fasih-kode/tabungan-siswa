const { test } = require('node:test');
const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const vm = require('node:vm');
const source = readFileSync('static/js/instant-search.js', 'utf8');

// A small DOM/network adapter lets the real controller run with deterministic
// timers and out-of-order responses, without introducing a browser dependency.
class Element {
  constructor() { this.listeners = {}; this.attributes = {}; this.children = []; this.textContent = ''; this.dataset = {}; }
  addEventListener(name, fn) { (this.listeners[name] ??= []).push(fn); }
  emit(name, extras = {}) {
    const event = { target: this, button: 0, defaultPrevented: false, preventDefault() { this.defaultPrevented = true; }, ...extras };
    for (const fn of this.listeners[name] ?? []) fn(event);
    return event;
  }
  setAttribute(name, value) { this.attributes[name] = value; }
  hasAttribute(name) { return Object.hasOwn(this.attributes, name); }
  toggleAttribute(name, enabled) { if (enabled) this.attributes[name] = ''; else delete this.attributes[name]; }
  append(child) { this.children.push(child); }
}
class Input extends Element {
  constructor(name, value) { super(); this.name = name; this.value = value; this.valid = true; }
  checkValidity() { return this.valid; }
}
class Select extends Element {
  constructor(name, value) {
    super(); this.name = name; this.value = value; this.disabled = false;
    this.options = [{ cloneNode: () => ({ value: '' }) }]; this.innerHTML = '<option value="">All</option>';
  }
  replaceChildren(...options) { this.options = options; }
}
function form(fields) {
  const el = new Element(); el.method = 'get'; el.action = 'http://localhost/students'; el.elements = fields;
  fields.namedItem = (name) => fields.find((field) => field.name === name);
  el.querySelectorAll = () => fields.filter((field) => field instanceof Input);
  return el;
}
function region(formEl, rows = 'original rows', summaryText = 'Menampilkan 20 siswa') {
  const el = new Element(); el.id = 'student-directory';
  const results = new Element(); results.innerHTML = rows; results.contains = (link) => link.inResults;
  const status = new Element(); const summary = new Element(); summary.textContent = summaryText;
  el.querySelector = (selector) => ({
    'form[data-instant-filter]': formEl,
    '[data-instant-results]': results,
    '[data-instant-status]': status,
    '[data-instant-summary]': summary,
  })[selector];
  return Object.assign(el, { form: formEl, results, status, summary });
}
function setup() {
  const input = new Input('q', ''); const year = new Select('academic_year_id', 'year-old');
  year.dataset.resetFilter = 'class_id';
  const classes = new Select('class_id', 'class-old'); const statusFilter = new Select('status', 'ACTIVE');
  const currentForm = form([input, year, classes, statusFilter]); const current = region(currentForm);
  const location = { href: 'http://localhost/students?page=4&limit=5' };
  const history = { state: {}, changes: [], replaceState(state, _, url) { this.state = state; location.href = String(url); this.changes.push('replace'); }, pushState(state, _, url) { this.state = state; location.href = String(url); this.changes.push('push'); } };
  const pending = []; const timers = new Map(); let time = 0; let serial = 0;
  const window = new Element();
  const document = {
    querySelectorAll: () => [current],
    createElement: () => new Element(),
  };
  class FormData {
    constructor(formEl) { this.entries = formEl.elements.filter((field) => !field.disabled).map((field) => [field.name, field.value]); }
    [Symbol.iterator]() { return this.entries[Symbol.iterator](); }
  }
  class DOMParser {
    parseFromString(text) {
      const payload = JSON.parse(text);
      const nextForm = form(currentForm.elements.map((field) => {
        const next = field instanceof Select ? new Select(field.name, payload.values[field.name] ?? '') : new Input(field.name, payload.values[field.name] ?? '');
        next.innerHTML = '<option value="class-new">New class</option>';
        return next;
      }));
      const next = region(nextForm, payload.rows, payload.summary ?? 'Menampilkan 1 siswa');
      return { getElementById: () => next, querySelectorAll: () => [] };
    }
  }
  vm.runInNewContext(source, {
    document, window, location, history, URL, URLSearchParams, FormData, DOMParser,
    HTMLInputElement: Input, HTMLSelectElement: Select, AbortController,
    setTimeout(fn, delay) { const id = ++serial; timers.set(id, { fn, due: time + delay }); return id; },
    clearTimeout(id) { timers.delete(id); },
    fetch(url, options) { return new Promise((resolve, reject) => pending.push({ url: new URL(url), options, resolve, reject })); },
  });
  const advance = (ms) => {
    time += ms;
    for (const [id, timer] of [...timers]) if (timer.due <= time) { timers.delete(id); timer.fn(); }
  };
  const flush = () => new Promise((resolve) => setImmediate(resolve));
  const respond = async (index, { ok = true, rows = 'updated rows', summary, values } = {}) => {
    const req = pending[index];
    req.resolve({ ok, text: async () => JSON.stringify({ rows, summary, values: values ?? Object.fromEntries(req.url.searchParams) }) });
    await flush();
  };
  const type = (value) => { input.value = value; currentForm.emit('input', { target: input }); };
  return { ...current, region: current, input, year, classes, statusFilter, pending, location, history, window, advance, respond, flush, type };
}

test('typing is debounced, resets pagination, and keeps the search field mounted', async () => {
  const h = setup(); const input = h.input; input.selectionStart = 1;
  h.type('a'); h.advance(100); h.type('an'); h.advance(299);
  assert.equal(h.pending.length, 0);
  h.advance(1); assert.equal(h.pending.length, 1);
  const { url, options } = h.pending[0];
  assert.equal(url.searchParams.get('q'), 'an'); assert.equal(url.searchParams.get('page'), '1');
  assert.equal(url.searchParams.has('limit'), false); assert.equal(options.cache, 'no-store');
  await h.respond(0);
  assert.equal(h.form.elements.namedItem('q'), input);
  assert.equal(input.value, 'an'); assert.equal(input.selectionStart, 1);
  assert.equal(h.results.innerHTML, 'updated rows'); assert.equal(h.results.attributes['aria-busy'], 'false');
  assert.equal(h.summary.textContent, 'Menampilkan 1 siswa');
});

test('an old response cannot replace newer input, including during the debounce window', async () => {
  const h = setup(); h.type('old'); h.advance(300);
  h.type('latest'); assert.equal(h.pending[0].options.signal.aborted, true);
  await h.respond(0, { rows: 'stale rows' });
  assert.equal(h.results.innerHTML, 'original rows');
  h.advance(300); await h.respond(1, { rows: 'latest rows' });
  assert.equal(h.results.innerHTML, 'latest rows');
  assert.equal(new URL(h.location.href).searchParams.get('q'), 'latest');
});

test('changing year loads immediately and clears the old dependent class', async () => {
  const h = setup(); h.year.value = 'year-new';
  h.form.emit('change', { target: h.year });
  assert.equal(h.pending.length, 1); assert.equal(h.classes.value, ''); assert.equal(h.classes.disabled, true);
  assert.equal(h.pending[0].url.searchParams.get('academic_year_id'), 'year-new');
  assert.equal(h.pending[0].url.searchParams.has('class_id'), false);
  await h.respond(0);
  assert.equal(h.classes.disabled, false); assert.match(h.classes.innerHTML, /class-new/);
  assert.deepEqual(h.history.changes, ['push']);
});

test('composition waits for completed text; Enter uses asynchronous search', async () => {
  const h = setup(); h.form.emit('compositionstart'); h.type('文字'); h.advance(500);
  assert.equal(h.pending.length, 0);
  h.form.emit('compositionend'); h.advance(300); assert.equal(h.pending.length, 1);
  await h.respond(0);
  const event = h.form.emit('submit'); assert.equal(event.defaultPrevented, true);
  assert.equal(h.pending.length, 2);
});

test('failed requests preserve results and URL, with an asynchronous retry', async () => {
  const h = setup(); h.type('Ani'); h.advance(300); const originalURL = h.location.href;
  await h.respond(0, { ok: false });
  assert.equal(h.results.innerHTML, 'original rows'); assert.equal(h.location.href, originalURL);
  assert.equal(h.status.hasAttribute('data-error'), true);
  h.status.children.at(-1).emit('click'); assert.equal(h.pending.length, 2);
  await h.respond(1); assert.equal(h.status.hasAttribute('data-error'), false);
});

test('pagination stays asynchronous and Back restores the controls from server results', async () => {
  const h = setup();
  const link = new Element(); link.inResults = true; link.href = 'http://localhost/students?q=Ani&page=2&limit=5';
  const click = h.region.emit('click', { target: { closest: () => link } });
  assert.equal(click.defaultPrevented, true); assert.equal(h.pending[0].url.searchParams.get('page'), '2');
  await h.respond(0);
  assert.deepEqual(h.history.changes, ['push']);
  h.location.href = 'http://localhost/students?q=Budi&page=1';
  h.window.emit('popstate'); await h.respond(1);
  assert.equal(h.input.value, 'Budi'); assert.deepEqual(h.history.changes, ['push']);
  link.href = 'http://localhost/students/some-student';
  assert.equal(h.region.emit('click', { target: { closest: () => link } }).defaultPrevented, false);
});

test('incomplete patterned filters do not issue invalid requests', () => {
  const h = setup(); h.input.valid = false; h.type('incomplete-uuid'); h.advance(300);
  assert.equal(h.pending.length, 0); assert.match(h.status.textContent, /Lengkapi/);
});
