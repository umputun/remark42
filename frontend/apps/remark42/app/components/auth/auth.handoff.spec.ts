import { createHash, webcrypto } from 'crypto';
import { TextEncoder as NodeTextEncoder } from 'util';

import { handoffStartURL, handoffTxn, HANDOFF_MESSAGE, prepareHandoff, takeHandoff } from './auth.handoff';

describe('handoffStartURL', () => {
  it('turns the login the widget links to into the start page, with its own from and site', () => {
    const start = new URL(
      handoffStartURL('https://remark42.example.com/auth/github/login?from=https%3A%2F%2Fx%2F%3FselfClose&site=s', 'c')!
    );
    expect(start.origin).toBe('https://remark42.example.com');
    expect(start.pathname).toBe('/auth/handoff/start');
    expect(Object.fromEntries(start.searchParams)).toEqual({
      from: 'https://x/?selfClose',
      site: 's',
      provider: 'github',
      challenge: 'c',
    });
  });

  it('leaves a link that is not a provider login alone', () => {
    expect(handoffStartURL('https://remark42.example.com/auth/github/callback', 'c')).toBeNull();
    expect(handoffStartURL('https://remark42.example.com/web/iframe.html', 'c')).toBeNull();
  });
});

describe('handoffTxn', () => {
  it.each([
    [{ type: HANDOFF_MESSAGE, txn: 'a'.repeat(32) }, 'a'.repeat(32)],
    [{ type: HANDOFF_MESSAGE, txn: 'A'.repeat(32) }, null],
    [{ type: HANDOFF_MESSAGE, txn: 'a'.repeat(31) }, null],
    [{ type: 'other', txn: 'a'.repeat(32) }, null],
    ['remark42-handoff', null],
    [null, null],
  ])('reads %j as %j', (data, txn) => {
    expect(handoffTxn(data)).toBe(txn);
  });
});

describe('prepareHandoff', () => {
  const original = window.crypto;

  // jsdom has neither; a browser over https has both
  beforeAll(() => {
    Object.defineProperty(window, 'crypto', { value: webcrypto, configurable: true });
    Object.defineProperty(globalThis, 'TextEncoder', { value: NodeTextEncoder, configurable: true });
  });

  afterAll(() => {
    Object.defineProperty(window, 'crypto', { value: original, configurable: true });
    Object.defineProperty(globalThis, 'TextEncoder', { value: undefined, configurable: true });
  });

  /** the hash is made asynchronously, in a few turns of the event loop */
  async function ready() {
    for (let i = 0; i < 50; i++) {
      await new Promise((resolve) => setTimeout(resolve, 0));
    }
  }

  it('makes a verifier and its S256 challenge ahead of the click, a new one each time', async () => {
    prepareHandoff();
    await ready();
    const first = takeHandoff();
    expect(first).not.toBeNull();
    expect(first!.verifier).toMatch(/^[A-Za-z0-9_-]{43}$/);
    expect(first!.challenge).toBe(createHash('sha256').update(first!.verifier).digest('base64url'));

    await ready();
    const second = takeHandoff();
    expect(second).not.toBeNull();
    expect(second!.verifier).toMatch(/^[A-Za-z0-9_-]{43}$/);
    expect(second!.verifier).not.toBe(first!.verifier);
  });

  it('has nothing to give when none is ready yet', () => {
    takeHandoff();
    expect(takeHandoff()).toBeNull();
  });
});
