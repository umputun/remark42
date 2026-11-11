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

  it('dispatches REMARK42::ready on a page without the root node', async () => {
    // a single-page site loads the script on a route without comments and waits for the event to
    // call createInstance once a route with the node is shown
    document.body.innerHTML = '';
    jest.resetModules();
    // earlier tests leave createInstance on the shared window; the listener must find this import's own
    Reflect.deleteProperty(window, 'REMARK42');
    const ready = jest.fn(() => typeof window.REMARK42?.createInstance);
    window.addEventListener('REMARK42::ready', ready);

    await expect(import('./embed')).rejects.toThrow("Remark42: Can't find root node.");

    window.removeEventListener('REMARK42::ready', ready);
    expect(ready).toHaveBeenCalledTimes(1);
    expect(ready).toHaveReturnedWith('function');
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

  it('takes an open profile away on destroy, with the widget its sign-out is addressed to', async () => {
    const root = await mount();
    const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;

    window.dispatchEvent(
      new MessageEvent('message', { data: { profile: { id: 'reader', current: '1' } }, source: iframe.contentWindow })
    );
    expect(document.querySelectorAll('iframe')).toHaveLength(2);
    expect(document.body.style.overflow).toBe('hidden');
    // the dimmed layer over the whole page that the profile's frame sits in
    const overlay = document.querySelector('iframe[src*="page=profile"]')!.parentElement!;
    expect(overlay.style.display).toBe('block');

    window.REMARK42.destroy!();

    expect(document.querySelectorAll('iframe')).toHaveLength(0);
    expect(overlay.style.display).toBe('none');
    expect(document.body.style.overflow).toBe('');
  });

  it('resizes for its own iframe', async () => {
    const root = await mount();
    const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;

    window.dispatchEvent(new MessageEvent('message', { data: { height: 500 }, source: iframe.contentWindow }));

    expect(iframe.style.height).toBe('500px');
  });

  it('scrolls to a comment only after a frame is drawn at the height sent with it', async () => {
    const root = await mount();
    const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;
    const scrollTo = jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
    jest.spyOn(iframe, 'getBoundingClientRect').mockReturnValue({ top: 150 } as DOMRect);

    // scrolling in the frame that applies the height counts the page footer as shifted when the
    // scroll brings it into view, which it does for a comment near the end of the thread
    window.dispatchEvent(new MessageEvent('message', { data: { height: 5000 }, source: iframe.contentWindow }));
    window.dispatchEvent(new MessageEvent('message', { data: { scrollTo: 700 }, source: iframe.contentWindow }));

    expect(iframe.style.height).toBe('5000px');
    expect(scrollTo).not.toHaveBeenCalled();

    jest.advanceTimersToNextFrame();
    expect(scrollTo).not.toHaveBeenCalled();

    jest.advanceTimersToNextFrame();
    // the position is within the iframe, so the page adds where the iframe is
    expect(scrollTo).toHaveBeenCalledTimes(1);
    expect(scrollTo).toHaveBeenCalledWith(0, 850);

    scrollTo.mockRestore();
  });

  it('ignores a message with no source after the host page removed the widget', async () => {
    const root = await mount();
    const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;
    const scrollTo = jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
    iframe.style.height = '100px';

    // a host page that drops the root without destroy() leaves the listener attached, and a frame
    // removed right after posting delivers a message with no source. Browsers give an iframe
    // outside the page no contentWindow; jsdom keeps it, so it is taken away here
    root.remove();
    Object.defineProperty(iframe, 'contentWindow', { value: null });

    window.dispatchEvent(
      new MessageEvent('message', {
        data: { height: 9000, scrollTo: 5000, profile: { id: 'anyone', name: 'any text' } },
        source: null,
      })
    );
    jest.advanceTimersToNextFrame();
    jest.advanceTimersToNextFrame();

    expect(iframe.style.height).toBe('100px');
    expect(scrollTo).not.toHaveBeenCalled();
    expect(document.querySelectorAll('iframe')).toHaveLength(0);

    scrollTo.mockRestore();
  });

  it('drops a scroll still waiting for its frames when the widget is destroyed', async () => {
    const root = await mount();
    const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;
    const scrollTo = jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);

    window.dispatchEvent(new MessageEvent('message', { data: { scrollTo: 700 }, source: iframe.contentWindow }));
    window.REMARK42.destroy!();
    jest.advanceTimersByTime(100);

    expect(scrollTo).not.toHaveBeenCalled();

    scrollTo.mockRestore();
  });

  it('scrolls to a comment at once in a hidden tab, which draws no frames', async () => {
    const root = await mount();
    const iframe = root.querySelector<HTMLIFrameElement>(MARKED)!;
    const scrollTo = jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
    const visibility = jest.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');

    window.dispatchEvent(new MessageEvent('message', { data: { scrollTo: 700 }, source: iframe.contentWindow }));

    expect(scrollTo).toHaveBeenCalledTimes(1);

    visibility.mockRestore();
    scrollTo.mockRestore();
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

    // the page scrolls a couple of frames after the message, so let those pass before looking
    jest.advanceTimersByTime(100);

    expect(iframe.style.height).toBe('100px');
    expect(scrollTo).not.toHaveBeenCalled();
    expect(document.querySelectorAll('iframe')).toHaveLength(2);

    scrollTo.mockRestore();
  });

  describe('node', () => {
    afterEach(() => {
      delete window.remark_config.node;
    });

    it('mounts into the given element instead of the one with the default id', async () => {
      const root = await mount();
      const node = document.createElement('div');
      document.body.appendChild(node);

      window.REMARK42.createInstance({ ...window.remark_config, node });

      expect(node.querySelectorAll(`:scope > ${MARKED}`)).toHaveLength(1);
    });

    it('does not need the element with the default id', async () => {
      document.body.innerHTML = '<section id="spa"></section>';
      const node = document.getElementById('spa')!;
      window.remark_config.node = node;
      jest.resetModules();
      await import('./embed');

      expect(node.querySelectorAll(`:scope > ${MARKED}`)).toHaveLength(1);
    });

    it('keeps the element out of the iframe URL', async () => {
      const root = await mount();
      const node = document.createElement('div');
      document.body.appendChild(node);

      window.REMARK42.createInstance({ ...window.remark_config, node });

      const src = new URL(node.querySelector(MARKED)!.getAttribute('src')!);

      expect(src.searchParams.get('site_id')).toBe('remark');
      expect(src.searchParams.has('node')).toBe(false);
      expect(root.querySelectorAll(MARKED)).toHaveLength(1);
    });

    it('rejects a value that is not an element', async () => {
      await mount();

      expect(() =>
        window.REMARK42.createInstance({ ...window.remark_config, node: '#remark42' as unknown as HTMLElement })
      ).toThrow('Remark42: `node` must be a DOM element.');
    });

    it('falls back to the default id when no element is given', async () => {
      const root = await mount();
      root.querySelector(MARKED)!.remove();

      window.REMARK42.createInstance({ ...window.remark_config, node: undefined });

      expect(root.querySelectorAll(`:scope > ${MARKED}`)).toHaveLength(1);
    });
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
