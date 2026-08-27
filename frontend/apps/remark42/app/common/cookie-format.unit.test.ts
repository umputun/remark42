import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { authCookieOptions, readCookie, serializeCookie } from './cookie-format.ts';

/**
 * The attributes decide whether the cookie is delivered at all, and the arrangement they exist for
 * -- the widget embedded on another origin, over https -- is the one no local browser has. Both
 * sides of that branch are reachable here because the third-party flag is an argument.
 */
describe('authCookieOptions', () => {
  it('keeps the cookie off cross-site requests when the widget shares its page origin', () => {
    assert.deepEqual(authCookieOptions(false, true), { path: '/', sameSite: 'Strict', secure: true });
  });

  // Strict is never sent from a third-party frame: SameSite is judged against the top-level site,
  // not the request's own origin. None needs Secure, and Partitioned is what survives third-party
  // cookie blocking
  it('asks for delivery in a third-party frame when the page is on another origin', () => {
    assert.deepEqual(authCookieOptions(true, true), {
      path: '/',
      sameSite: 'None',
      secure: true,
      partitioned: true,
    });
  });

  // SameSite=None without Secure is rejected outright, which loses the cookie altogether
  it('does not claim attributes it cannot honour over http', () => {
    assert.deepEqual(authCookieOptions(true, false), { path: '/', sameSite: 'Strict', secure: false });
  });

  it('stays strict over http on a shared origin', () => {
    assert.deepEqual(authCookieOptions(false, false), { path: '/', sameSite: 'Strict', secure: false });
  });
});

describe('serializeCookie', () => {
  it('writes the name undecorated', () => {
    // a __Host- prefix stores the value under a name neither the backend nor the fetcher asks
    // for, and only on https, so it is invisible until deployment
    const cookie = serializeCookie('XSRF-TOKEN', 'jti-value');

    assert.equal(cookie, 'XSRF-TOKEN=jti-value');
    assert.equal(cookie.includes('__Host-'), false);
  });

  it('percent-encodes a value that would otherwise break the string apart', () => {
    assert.equal(serializeCookie('JWT', 'a;b c'), 'JWT=a%3Bb%20c');
  });

  it('writes a true boolean attribute as the flag alone', () => {
    assert.equal(serializeCookie('a', 'b', { secure: true }), 'a=b; secure');
  });

  // `secure=false` reads to a browser as the attribute being present, which is the opposite of
  // what it says
  it('omits a false boolean attribute instead of writing it as a value', () => {
    assert.equal(serializeCookie('a', 'b', { secure: false }), 'a=b');
  });

  it('carries the full third-party attribute set', () => {
    const cookie = serializeCookie('JWT', 'v', authCookieOptions(true, true));

    assert.equal(cookie, 'JWT=v; path=/; sameSite=None; secure; partitioned');
  });

  it('converts a Date expiry to the RFC-1123 form browsers parse', () => {
    assert.equal(serializeCookie('a', 'b', { expires: new Date(0) }), 'a=b; expires=Thu, 01 Jan 1970 00:00:00 GMT');
  });

  it('treats a numeric expiry as seconds from now', () => {
    const realNow = Date.now;

    Date.now = () => 0;
    try {
      assert.equal(serializeCookie('a', 'b', { expires: 60 }), 'a=b; expires=Thu, 01 Jan 1970 00:01:00 GMT');
    } finally {
      Date.now = realNow;
    }
  });

  it('passes a string expiry through unchanged', () => {
    const expires = 'Thu, 01 Jan 1970 00:00:00 GMT';

    assert.equal(serializeCookie('a', 'b', { expires }), `a=b; expires=${expires}`);
  });

  // clearing a cookie only works when every attribute matches the ones it was set with, and a
  // caller that reused its options object would otherwise find `expires` rewritten under it
  it('does not rewrite the options it was given', () => {
    const options = { expires: 60 };

    serializeCookie('a', 'b', options);
    assert.equal(options.expires, 60);
  });
});

describe('readCookie', () => {
  it('finds a cookie among others', () => {
    assert.equal(readCookie('a=1; JWT=token; b=2', 'JWT'), 'token');
  });

  it('finds the first cookie in the string', () => {
    assert.equal(readCookie('JWT=token; a=1', 'JWT'), 'token');
  });

  it('returns nothing when the cookie is absent', () => {
    assert.equal(readCookie('a=1', 'JWT'), undefined);
  });

  it('returns nothing from an empty cookie string', () => {
    assert.equal(readCookie('', 'JWT'), undefined);
  });

  it('decodes the value', () => {
    assert.equal(readCookie('JWT=a%3Bb%20c', 'JWT'), 'a;b c');
  });

  it('reads a present but empty cookie as an empty string', () => {
    assert.equal(readCookie('JWT=; a=1', 'JWT'), '');
  });

  // the name is embedded in a regex, so an unescaped metacharacter would match the wrong cookie
  it('does not let a name suffix match a longer cookie name', () => {
    assert.equal(readCookie('XSRF-TOKEN=x', 'TOKEN'), undefined);
  });

  // the decoy has to come first, or an unescaped `.` still finds the right cookie by luck and
  // the case passes against the very defect it names
  it('escapes regex metacharacters in the name', () => {
    assert.equal(readCookie('axb=2; a.b=1', 'a.b'), '1');
  });

  // `decodeURIComponent` throws on a malformed escape, and the caller reads this while building a
  // request, outside any handler of its own: a throw there rejects every API call with an error the
  // catalogue does not recognize, for as long as the cookie is there
  it('hands back a malformed percent escape raw instead of throwing', () => {
    assert.equal(readCookie('JWT=%E0%A4%A; a=1', 'JWT'), '%E0%A4%A');
  });

  it('still reads the cookies beside a malformed one', () => {
    assert.equal(readCookie('JWT=%E0%A4%A; a=1', 'a'), '1');
  });

  it('round-trips what serializeCookie wrote', () => {
    const cookie = serializeCookie('XSRF-TOKEN', 'jti;value');

    assert.equal(readCookie(cookie.split(';')[0], 'XSRF-TOKEN'), 'jti;value');
  });
});
