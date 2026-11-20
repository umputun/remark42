import type * as Settings from './settings';

/**
 * How max_shown_comments reaches the widget once parsed: what a value is accepted as is covered by
 * `max-shown-comments.unit.test.ts`, and how the root uses the result by `root.spec.tsx`. What is
 * left is the default, which is the only part that depends on the device.
 */
describe('maxShownComments', () => {
  async function loadMaxShownComments(search: string, isMobile: boolean) {
    window.history.replaceState(null, '', `/web/iframe.html${search}`);
    let settings: typeof Settings | undefined;

    await jest.isolateModulesAsync(async () => {
      jest.doMock('./constants', () => ({ ...jest.requireActual('./constants'), IS_MOBILE: isMobile }));
      settings = await import('./settings');
    });

    return settings!.maxShownComments;
  }

  afterEach(() => {
    jest.dontMock('./constants');
    window.history.replaceState(null, '', '/');
  });

  it('applies a configured value on desktop and on mobile alike', async () => {
    expect(await loadMaxShownComments('?max_shown_comments=3', false)).toBe(3);
    expect(await loadMaxShownComments('?max_shown_comments=3', true)).toBe(3);
  });

  it('shows ten at first on mobile and every comment on desktop when nothing is configured', async () => {
    expect(await loadMaxShownComments('', true)).toBe(10);
    expect(await loadMaxShownComments('', false)).toBe(Infinity);
  });

  it('treats a value that is not a positive whole number as not configured', async () => {
    expect(await loadMaxShownComments('?max_shown_comments=0', true)).toBe(10);
    expect(await loadMaxShownComments('?max_shown_comments=0', false)).toBe(Infinity);
  });
});
