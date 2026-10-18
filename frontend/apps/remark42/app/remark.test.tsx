process.env.REMARK_NODE = 'remark42';
// webpack defines it in the bundle; init assigns it before anything else
globalThis.__webpack_public_path__ = '';

// keeps init suspended at its first await, which is the window a theme message used to be lost in
jest.mock('utils/loadLocale', () => ({ loadLocale: () => new Promise(() => undefined) }));

// every start loads a fresh copy of the module, and each would leave its listener on window to
// react to the later tests' messages too
let listeners: Array<Parameters<typeof window.addEventListener>> = [];

async function start() {
  document.body.innerHTML = '<div id="remark42"></div>';
  jest.resetModules();
  const addEventListener = jest.spyOn(window, 'addEventListener');
  await import('./remark');
  listeners = [...addEventListener.mock.calls];
  addEventListener.mockRestore();
  // the instance remark.tsx dispatches to: both imports follow the same reset
  const { store } = await import('store');

  return store;
}

describe('remark', () => {
  afterEach(() => {
    listeners.forEach((args) => window.removeEventListener(...args));
    listeners = [];
    delete window.remark_config.theme;
    document.body.className = '';
    document.body.innerHTML = '';
    document.documentElement.style.colorScheme = '';
  });

  it('takes the theme iframe.ejs kept over the one in the URL', async () => {
    // what the inline listener in iframe.ejs leaves for a theme posted before the bundle ran
    window.remark_config.theme = 'dark';

    const store = await start();

    expect(document.body.classList.contains('dark')).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe('dark');
    expect(store.getState().theme).toBe('dark');
  });

  it('applies a theme posted while the locale is still loading', async () => {
    const store = await start();
    expect(store.getState().theme).toBe('light');

    window.dispatchEvent(new MessageEvent('message', { data: { theme: 'dark' }, source: window.parent }));

    expect(document.body.classList.contains('dark')).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe('dark');
    expect(store.getState().theme).toBe('dark');
  });

  it('ignores a theme posted by anything but its parent', async () => {
    const store = await start();
    const foreign = document.createElement('iframe');
    document.body.appendChild(foreign);

    window.dispatchEvent(new MessageEvent('message', { data: { theme: 'dark' }, source: foreign.contentWindow }));

    expect(document.body.classList.contains('dark')).toBe(false);
    expect(store.getState().theme).toBe('light');
  });
});
