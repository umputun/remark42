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

  describe('theme', () => {
    afterEach(() => {
      delete window.remark_config.theme;
    });

    it('posts the current theme once the frame reports inited', async () => {
      const root = await mount();
      const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;

      // made while the document is still loading, so the message it posts reaches no listener
      window.REMARK42.changeTheme!('dark');
      const postMessage = jest.spyOn(iframe.contentWindow!, 'postMessage');

      window.dispatchEvent(new MessageEvent('message', { data: { inited: true }, source: iframe.contentWindow }));

      expect(postMessage).toHaveBeenCalledWith({ theme: 'dark' }, '*');
    });

    it('posts no theme on inited when none was set', async () => {
      const root = await mount();
      const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;
      const postMessage = jest.spyOn(iframe.contentWindow!, 'postMessage');

      window.dispatchEvent(new MessageEvent('message', { data: { inited: true }, source: iframe.contentWindow }));

      expect(postMessage).not.toHaveBeenCalledWith(expect.objectContaining({ theme: expect.anything() }), '*');
    });

    describe('auto', () => {
      let systemDark = false;
      const schemeListeners = new Set<() => void>();

      function setSystemScheme(dark: boolean) {
        systemDark = dark;
        schemeListeners.forEach((listener) => listener());
      }

      beforeEach(() => {
        systemDark = false;
        schemeListeners.clear();
        window.matchMedia = jest.fn((query: string) => ({
          media: query,
          get matches() {
            return query === '(prefers-color-scheme: dark)' && systemDark;
          },
          addEventListener: (_: string, listener: () => void) => schemeListeners.add(listener),
          removeEventListener: (_: string, listener: () => void) => schemeListeners.delete(listener),
        })) as unknown as typeof window.matchMedia;
      });

      afterEach(() => {
        // jsdom has no matchMedia of its own
        delete (window as Partial<Window>).matchMedia;
      });

      it('builds the iframe in the theme the system prefers', async () => {
        window.remark_config.theme = 'auto';
        systemDark = true;
        const root = await mount();
        const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;

        expect(new URL(iframe.src).searchParams.get('theme')).toBe('dark');
        expect(iframe.style.colorScheme).toBe('dark');
        // the host page's config keeps the setting, so the widget goes on following the system
        expect(window.remark_config.theme).toBe('auto');
      });

      it('posts the resolved theme once the frame reports inited', async () => {
        window.remark_config.theme = 'auto';
        systemDark = true;
        const root = await mount();
        const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;
        const postMessage = jest.spyOn(iframe.contentWindow!, 'postMessage');

        window.dispatchEvent(new MessageEvent('message', { data: { inited: true }, source: iframe.contentWindow }));

        expect(postMessage).toHaveBeenCalledWith({ theme: 'dark' }, '*');
      });

      it('switches along with the system preference', async () => {
        window.remark_config.theme = 'auto';
        const root = await mount();
        const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;
        const postMessage = jest.spyOn(iframe.contentWindow!, 'postMessage');

        setSystemScheme(true);

        expect(iframe.style.colorScheme).toBe('dark');
        expect(postMessage).toHaveBeenLastCalledWith({ theme: 'dark' }, '*');

        setSystemScheme(false);

        expect(iframe.style.colorScheme).toBe('light');
        expect(postMessage).toHaveBeenLastCalledWith({ theme: 'light' }, '*');
      });

      it('starts following the system preference on changeTheme', async () => {
        window.remark_config.theme = 'light';
        systemDark = true;
        const root = await mount();
        const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;
        const postMessage = jest.spyOn(iframe.contentWindow!, 'postMessage');

        window.REMARK42.changeTheme!('auto');

        expect(iframe.style.colorScheme).toBe('dark');
        expect(postMessage).toHaveBeenLastCalledWith({ theme: 'dark' }, '*');

        setSystemScheme(false);

        expect(postMessage).toHaveBeenLastCalledWith({ theme: 'light' }, '*');
      });

      it('stops following the system preference once a fixed theme is set', async () => {
        window.remark_config.theme = 'auto';
        const root = await mount();
        const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;

        window.REMARK42.changeTheme!('light');
        const postMessage = jest.spyOn(iframe.contentWindow!, 'postMessage');
        setSystemScheme(true);

        expect(iframe.style.colorScheme).toBe('light');
        expect(postMessage).not.toHaveBeenCalled();
      });

      it('opens the profile in the resolved theme', async () => {
        window.remark_config.theme = 'auto';
        systemDark = true;
        const root = await mount();
        const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;

        window.dispatchEvent(
          new MessageEvent('message', { data: { profile: { id: 'user' } }, source: iframe.contentWindow })
        );

        const profile = Array.from(document.querySelectorAll('iframe')).find((frame) => frame !== iframe)!;
        expect(new URL(profile.src).searchParams.get('theme')).toBe('dark');
      });

      it('stops listening to the system preference on destroy', async () => {
        window.remark_config.theme = 'auto';
        await mount();

        window.REMARK42.destroy!();

        expect(schemeListeners.size).toBe(0);
      });

      it('leaves a single listener when createInstance is called again', async () => {
        window.remark_config.theme = 'auto';
        await mount();

        window.REMARK42.createInstance(window.remark_config);

        expect(schemeListeners.size).toBe(1);
      });
    });
  });
});
