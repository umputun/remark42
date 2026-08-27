import assert from 'node:assert/strict';
import { afterEach, describe, it } from 'node:test';

import { resolveBaseUrl } from './base-url.ts';

/**
 * The host comes from the embedding page, so it is attacker-controlled wherever that page is, and
 * whatever survives this check becomes the origin every request and the iframe src are built from.
 * The jest suite reaches one branch of it by running in a jsdom pinned to an https url; here the
 * page protocol is just an argument, so every combination is one line.
 */
function captureErrors() {
  const real = console.error;
  const seen: string[] = [];

  console.error = (...args: unknown[]) => {
    seen.push(String(args[0]));
  };

  return { seen, restore: () => (console.error = real) };
}

let capture: ReturnType<typeof captureErrors> | undefined;

afterEach(() => {
  capture?.restore();
  capture = undefined;
});

describe('resolveBaseUrl', () => {
  it('accepts an http host on an http page', () => {
    capture = captureErrors();

    assert.equal(resolveBaseUrl('http://example.com', 'http:'), 'http://example.com');
    assert.deepEqual(capture.seen, []);
  });

  it('accepts an https host on an https page', () => {
    capture = captureErrors();

    assert.equal(resolveBaseUrl('https://example.com', 'https:'), 'https://example.com');
    assert.deepEqual(capture.seen, []);
  });

  it('keeps the host exactly as given, path and port included', () => {
    capture = captureErrors();

    assert.equal(resolveBaseUrl('http://example.com:8080/sub', 'http:'), 'http://example.com:8080/sub');
  });

  it('says so when the widget was not configured at all', () => {
    assert.throws(() => resolveBaseUrl(undefined, 'http:'), /remark_config.host wasn't configured/);
  });

  it('treats an empty host as unconfigured instead of as a url', () => {
    assert.throws(() => resolveBaseUrl('', 'http:'), /remark_config.host wasn't configured/);
  });

  // mixed content is the browser's call, not this function's: refusing here would take the widget
  // down over an arrangement that may well be intended
  it('reports a protocol mismatch but still returns the host', () => {
    capture = captureErrors();

    assert.equal(resolveBaseUrl('https://example.com', 'http:'), 'https://example.com');
    assert.deepEqual(capture.seen, ['Remark42: Protocol mismatch.']);
  });

  it('reports the mismatch in the other direction too', () => {
    capture = captureErrors();

    assert.equal(resolveBaseUrl('http://example.com', 'https:'), 'http://example.com');
    assert.deepEqual(capture.seen, ['Remark42: Protocol mismatch.']);
  });

  // this is the guard that matters: whatever comes back is used as the widget's own origin, so a
  // scheme that executes would run as the page
  it('refuses a javascript url', () => {
    capture = captureErrors();

    // eslint-disable-next-line no-script-url -- the scheme is the input under test
    assert.throws(() => resolveBaseUrl('javascript:alert(1)', 'http:'), /Invalid host URL/);
    assert.ok(capture.seen.includes('Remark42: Wrong protocol in host URL.'));
  });

  it('refuses a data url', () => {
    capture = captureErrors();

    assert.throws(() => resolveBaseUrl('data:text/html,<script>alert(1)</script>', 'http:'), /Invalid host URL/);
  });

  it('refuses other schemes that are not http', () => {
    capture = captureErrors();

    for (const host of ['ftp://example.com', 'file:///etc/passwd', 'ws://example.com', 'blob:http://example.com/x']) {
      assert.throws(() => resolveBaseUrl(host, 'http:'), /Invalid host URL/, host);
    }
  });

  it('refuses a protocol-relative host, which has no protocol to check', () => {
    capture = captureErrors();

    assert.throws(() => resolveBaseUrl('//example.com', 'http:'), /Invalid host URL/);
  });

  it('refuses a bare hostname', () => {
    capture = captureErrors();

    assert.throws(() => resolveBaseUrl('example.com', 'http:'), /Invalid host URL/);
  });

  it('refuses something that is not a url at all', () => {
    capture = captureErrors();

    assert.throws(() => resolveBaseUrl('not a url', 'http:'), /Invalid host URL/);
  });

  // the check compares exactly and not by prefix: a scheme whose name merely begins with those
  // four letters is not http, and whatever survives here becomes the widget's own origin
  it('refuses a scheme merely starting with http', () => {
    capture = captureErrors();

    assert.throws(() => resolveBaseUrl('httpx://example.com', 'http:'), /Invalid host URL/);
    assert.throws(() => resolveBaseUrl('https-x://example.com', 'http:'), /Invalid host URL/);
  });
});
