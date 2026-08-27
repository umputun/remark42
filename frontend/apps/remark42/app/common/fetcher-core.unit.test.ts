import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { authHeaders, clockSkewMs, jwtPayload, requestBody, requestURL } from './fetcher-core.ts';

/** Swaps console.error out so a case that expects a report can assert it, and print nothing. */
function captureErrors() {
  const real = console.error;
  const seen: unknown[][] = [];

  console.error = (...args: unknown[]) => {
    seen.push(args);
  };

  return { seen, restore: () => (console.error = real) };
}

/**
 * base64url is not base64: `-` stands in for `+` and `_` for `/`. The payload below is chosen to
 * carry both, so a decoder that skips the substitution cannot pass by luck.
 */
const B64URL_TOKEN = 'header.eyJqdGkiOiI2MDN-MWU_NnQifQ.signature';

describe('jwtPayload', () => {
  it('decodes the payload', () => {
    assert.deepEqual(jwtPayload('h.eyJqdGkiOiJhYmMiLCJleHAiOjF9.s'), { jti: 'abc', exp: 1 });
  });

  it('substitutes the base64url alphabet', () => {
    assert.deepEqual(jwtPayload(B64URL_TOKEN), { jti: '603~1e?6t' });
  });

  // returning null instead of throwing is what lets a request whose token this widget cannot read
  // still be sent, which is the case where the backend is the only one that needs to read it
  it('returns null when there is no payload segment', () => {
    assert.equal(jwtPayload('nodots'), null);
  });

  it('returns null for an empty payload segment', () => {
    assert.equal(jwtPayload('h..s'), null);
  });

  it('returns null when the payload is not base64', () => {
    assert.equal(jwtPayload('h.!!!!.s'), null);
  });

  it('returns null when the decoded payload is not JSON', () => {
    assert.equal(jwtPayload('h.bm90IGpzb24.s'), null);
  });

  it('returns null for an empty token', () => {
    assert.equal(jwtPayload(''), null);
  });

  // a token this widget cannot decode leaves the auth cookies unwritten, and that path is where a
  // base64url regression shows first, so it has to leave a trace
  it('reports a payload it cannot decode', () => {
    const capture = captureErrors();

    try {
      assert.equal(jwtPayload('h.!!!!.s'), null);
      assert.equal(capture.seen.length, 1);
    } finally {
      capture.restore();
    }
  });
});

/**
 * The reading feeds the comment edit deadline, so a wrong answer either closes editing early or
 * leaves it open for good. The case that matters is the header being absent, which no browser test
 * can produce because the server always sends one.
 */
describe('clockSkewMs', () => {
  const hour = 60 * 60 * 1000;
  const day = 24 * hour;
  const serverSaid = 'Mon, 26 Aug 2026 12:00:00 GMT';
  const serverMs = Date.parse(serverSaid);

  it('reports how far the client clock is ahead', () => {
    assert.equal(clockSkewMs(serverSaid, serverMs + 5000, day), 5000);
  });

  it('reports a negative skew when the client is behind', () => {
    assert.equal(clockSkewMs(serverSaid, serverMs - 5000, day), -5000);
  });

  it('reports no skew when the clocks agree', () => {
    assert.equal(clockSkewMs(serverSaid, serverMs, day), 0);
  });

  // both guards reject a missing header, so this case cannot say which one did it. it pins the
  // answer; what pins the reason is that removing the isNaN guard fails all three of these
  it('rejects a missing header', () => {
    assert.equal(clockSkewMs(null, serverMs, day), null);
  });

  it('rejects an empty header', () => {
    assert.equal(clockSkewMs('', serverMs, day), null);
  });

  // Date.parse is lenient enough to turn junk into a date of its own accord, so the bound is what
  // stops a garbled header being believed
  it('rejects a header it cannot parse', () => {
    assert.equal(clockSkewMs('not a date', serverMs, day), null);
  });

  it('rejects a reading at the bound', () => {
    assert.equal(clockSkewMs(serverSaid, serverMs + day, day), null);
  });

  it('accepts a reading just inside the bound', () => {
    assert.equal(clockSkewMs(serverSaid, serverMs + day - 1, day), day - 1);
  });

  it('rejects a reading beyond the bound in either direction', () => {
    assert.equal(clockSkewMs(serverSaid, serverMs + 2 * day, day), null);
    assert.equal(clockSkewMs(serverSaid, serverMs - 2 * day, day), null);
  });
});

describe('requestURL', () => {
  it('adds the site every endpoint requires', () => {
    assert.equal(requestURL('http://x/api', '/find', {}, 'remark'), 'http://x/api/find?site=remark');
  });

  it('keeps the caller query beside it', () => {
    assert.equal(
      requestURL('http://x/api', '/find', { url: 'http://p/', sort: '-active' }, 'remark'),
      'http://x/api/find?site=remark&url=http%3A%2F%2Fp%2F&sort=-active'
    );
  });

  it('lets an explicit site win, which the cross-site calls rely on', () => {
    assert.equal(requestURL('http://x/api', '/find', { site: 'other' }, 'remark'), 'http://x/api/find?site=other');
  });

  it('renders a numeric value', () => {
    assert.equal(
      requestURL('http://x/api', '/vote/1', { vote: 1 }, 'remark'),
      'http://x/api/vote/1?site=remark&vote=1'
    );
  });

  // blockUser sends `ttl: undefined` for a permanent block. serialized, that is the literal string
  // "undefined", which the backend fails to parse and then silently treats as unlimited: the right
  // outcome by accident, and only for as long as it keeps swallowing the parse error
  it('omits an undefined value instead of sending the word', () => {
    const url = requestURL('http://x/api', '/user/1', { block: 1, ttl: undefined }, 'remark');

    assert.equal(url, 'http://x/api/user/1?site=remark&block=1');
    assert.equal(url.includes('undefined'), false);
  });

  it('encodes a value that would otherwise break the query apart', () => {
    assert.equal(requestURL('http://x/api', '/f', { q: 'a&b=c' }, 'remark'), 'http://x/api/f?site=remark&q=a%26b%3Dc');
  });

  it('encodes a site id with a reserved character in it', () => {
    assert.equal(requestURL('http://x/api', '/f', {}, 'a b&c'), 'http://x/api/f?site=a+b%26c');
  });
});

describe('requestBody', () => {
  // the browser writes a Content-Type naming the multipart boundary it generated; a header set
  // here would replace it with one naming no boundary, and the server could not parse the upload
  it('passes FormData through with no content type of its own', () => {
    const form = new FormData();
    const { body, headers } = requestBody(form, 'remark');

    assert.equal(body, form);
    assert.deepEqual(headers, {});
  });

  it('sends an object as json carrying the site', () => {
    const { body, headers } = requestBody({ text: 'hi' }, 'remark');

    assert.deepEqual(JSON.parse(body as string), { text: 'hi', site: 'remark' });
    assert.deepEqual(headers, { 'Content-Type': 'application/json' });
  });

  it('lets the configured site win over one in the body', () => {
    const { body } = requestBody({ site: 'other' }, 'remark');

    assert.deepEqual(JSON.parse(body as string), { site: 'remark' });
  });

  it('passes a string body through untouched', () => {
    assert.deepEqual(requestBody('raw', 'remark'), { body: 'raw', headers: {} });
  });

  it('treats null as no body, not as an object', () => {
    assert.deepEqual(requestBody(null, 'remark'), { body: null, headers: {} });
  });

  it('treats a missing body as no body', () => {
    assert.deepEqual(requestBody(undefined, 'remark'), { body: undefined, headers: {} });
  });
});

describe('authHeaders', () => {
  it('carries both tokens when both are held', () => {
    assert.deepEqual(authHeaders('jwt-value', 'xsrf-value'), {
      'X-JWT': 'jwt-value',
      'X-XSRF-TOKEN': 'xsrf-value',
    });
  });

  it('sends neither when neither is held', () => {
    assert.deepEqual(authHeaders(undefined, undefined), {});
  });

  it('omits the jwt header instead of sending it empty', () => {
    assert.deepEqual(authHeaders(undefined, 'xsrf-value'), { 'X-XSRF-TOKEN': 'xsrf-value' });
  });

  it('omits the xsrf header when there is no cookie', () => {
    assert.deepEqual(authHeaders('jwt-value', undefined), { 'X-JWT': 'jwt-value' });
  });

  // a `XSRF-TOKEN=;` cookie reads back as an empty string, and an empty header is what lighttpd
  // answers 400 to. the backend reads the header with Header.Get, which gives the same empty
  // string for a missing header, so omitting it loses nothing
  it('omits the xsrf header for a cookie that is present but empty', () => {
    assert.deepEqual(authHeaders(undefined, ''), {});
  });

  it('omits both for empty values', () => {
    assert.deepEqual(authHeaders('', ''), {});
  });
});
