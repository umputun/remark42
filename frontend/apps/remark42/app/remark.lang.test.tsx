// webpack defines it in the bundle; init assigns it before anything else
globalThis.__webpack_public_path__ = '';

let mockCatalogue: Record<string, string> = {};
jest.mock('utils/loadLocale', () => ({ loadLocale: () => Promise.resolve(mockCatalogue) }));
// init sets the language before it waits for the server's config, and stops there
jest.mock('common/api', () => ({
  ...jest.requireActual('common/api'),
  getConfig: () => new Promise(() => undefined),
}));

// settings reads the locale from the address once, when the module loads
async function startIn(locale: string) {
  window.history.replaceState(null, '', `/?locale=${locale}`);
  document.body.innerHTML = '<div id="remark42"></div>';
  document.documentElement.removeAttribute('lang');
  jest.resetModules();
  await import('./remark');
}

describe('the language of the widget document', () => {
  afterEach(() => {
    window.history.replaceState(null, '', '/');
    document.body.innerHTML = '';
  });

  it('is the tag of the locale whose catalogue it shows', async () => {
    mockCatalogue = { 'root.show-more': 'Показать ещё' };
    await startIn('ua');

    await new Promise((resolve) => setTimeout(resolve));
    expect(document.documentElement.lang).toBe('uk');
  });

  it('is English when the catalogue does not load', async () => {
    mockCatalogue = {};
    await startIn('ru');

    await new Promise((resolve) => setTimeout(resolve));
    expect(document.documentElement.lang).toBe('en');
  });
});
