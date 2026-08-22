process.env.REMARK_NODE = 'remark42';

const MARKER = 'data-remark42-iframe';
const MARKED = `iframe[${MARKER}]`;

async function mount(placeholder = '') {
  document.body.innerHTML = `<div id="remark42">${placeholder}</div>`;
  jest.resetModules();
  await import('./embed');

  return document.getElementById('remark42')!;
}

describe('embed', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
    document.body.innerHTML = '';
  });

  it('creates its iframe rather than adopting an element placeholder', async () => {
    const root = await mount('<noscript>Please enable JavaScript to view the comments.</noscript>');

    expect(root.querySelectorAll(MARKED)).toHaveLength(1);
  });

  it('creates its iframe rather than adopting an unmarked one', async () => {
    const root = await mount('<iframe title="placeholder"></iframe>');

    expect(root.querySelectorAll('iframe')).toHaveLength(2);
    expect(root.querySelectorAll(MARKED)).toHaveLength(1);
  });

  it('creates its iframe rather than adopting a marked one further down the tree', async () => {
    const root = await mount(`<div><iframe ${MARKER}></iframe></div>`);

    expect(root.querySelectorAll('iframe')).toHaveLength(2);
    expect(root.querySelectorAll(`:scope > ${MARKED}`)).toHaveLength(1);
  });

  it('reuses its own iframe on a second createInstance', async () => {
    const root = await mount();
    const first = root.querySelector(MARKED);

    window.REMARK42.createInstance(window.remark_config);

    expect(root.querySelectorAll('iframe')).toHaveLength(1);
    expect(root.querySelector(MARKED)).toBe(first);
  });

  it('does not stack listeners when createInstance is called again', async () => {
    const root = await mount();
    const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;

    // the second call reuses the iframe rather than building one, so without a teardown the first
    // call's handlers stay attached and both react to every message
    window.REMARK42.createInstance(window.remark_config);
    window.REMARK42.createInstance(window.remark_config);

    let resizes = 0;
    Object.defineProperty(iframe.style, 'height', {
      set() {
        resizes += 1;
      },
      get: () => '',
      configurable: true,
    });

    window.dispatchEvent(new MessageEvent('message', { data: { height: 500 }, source: iframe.contentWindow }));

    expect(resizes).toBe(1);
  });

  it('detaches every listener on destroy, however many instances were created', async () => {
    const root = await mount();
    const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;

    window.REMARK42.createInstance(window.remark_config);
    window.REMARK42.destroy!();

    let resized = false;
    Object.defineProperty(iframe.style, 'height', {
      set() {
        resized = true;
      },
      get: () => '',
      configurable: true,
    });

    window.dispatchEvent(new MessageEvent('message', { data: { height: 500 }, source: iframe.contentWindow }));

    expect(resized).toBe(false);
  });

  it('resizes for its own iframe', async () => {
    const root = await mount();
    const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;

    window.dispatchEvent(new MessageEvent('message', { data: { height: 500 }, source: iframe.contentWindow }));

    expect(iframe.style.height).toBe('500px');
  });

  it('ignores a message from a frame it does not own', async () => {
    const root = await mount();
    const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;
    iframe.style.height = '100px';

    // every frame on a page can reach window.parent, an advertisement among them, and the handler
    // resizes the widget, scrolls the page and opens the profile overlay
    const foreign = document.createElement('iframe');
    document.body.appendChild(foreign);
    const scrollTo = jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);

    window.dispatchEvent(
      new MessageEvent('message', {
        data: { height: 9000, scrollTo: 5000, profile: { id: 'anyone' } },
        source: foreign.contentWindow,
      })
    );

    expect(iframe.style.height).toBe('100px');
    expect(scrollTo).not.toHaveBeenCalled();
    expect(document.querySelectorAll('iframe')).toHaveLength(2);

    scrollTo.mockRestore();
  });
});
