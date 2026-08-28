import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { validToken, invalidToken } from '../__stubs__/jwt.ts';

import { isJwtExpired, parseJwt } from './jwt.ts';

/**
 * The token in __stubs__ expired in January 2020, so `isJwtExpired` answers true against a real
 * clock whatever it does with the claim. The expiry cases below pin `Date.now` on either side of
 * the token's own `exp` instead, which is the only way the comparison itself is under test.
 *
 * The failure case asserts the exception type instead of its message: `atob` is a host builtin and
 * every runtime words the failure differently -- jsdom, node and bun each have their own string --
 * so matching one of them would tie the test to the runner it first ran under.
 */
describe('parseJwt', () => {
  it('decodes the payload', () => {
    assert.deepEqual(parseJwt(validToken), {
      aud: 'remark',
      exp: 1579986982,
      handshake: { id: 'dev_user::asd@x101.pw' },
      iss: 'remark42',
      nbf: 1579985122,
    });
  });

  // the stub token's payload carries neither `-` nor `_`, so it cannot tell the substitution from
  // its absence. this one carries both, and plain base64 decodes it to nonsense
  it('substitutes the base64url alphabet', () => {
    const token =
      'header.eyJhdWQiOiJyZW1hcmsiLCJleHAiOjE1Nzk5ODY5ODIsImlzcyI6InJlbWFyazQyIiwibmJmIjoxNTc5OTg1MTIyLCJoYW5kc2hha2UiOnsiaWQiOiJkZXZfdXNlcjo6YXNkQHgxMDEucHcifSwianRpIjoiYT9iPmN-ZCJ9.sig';

    assert.equal(parseJwt<{ exp: number; jti: string }>(token).jti, 'a?b>c~d');
  });

  it('throws on a truncated token', () => {
    assert.throws(() => parseJwt(invalidToken), DOMException);
  });

  it('throws when there is no payload segment at all', () => {
    assert.throws(() => parseJwt('not-a-token'));
  });
});

describe('isJwtExpired', () => {
  // exp is in seconds, Date.now in milliseconds, and getting that conversion wrong is the mistake
  // worth catching: a token would then read as valid for a thousand times its lifetime
  it('is not expired a second before exp', (t) => {
    t.mock.method(Date, 'now', () => 1579986981 * 1000);
    assert.equal(isJwtExpired(validToken), false);
  });

  it('is not expired exactly at exp', (t) => {
    t.mock.method(Date, 'now', () => 1579986982 * 1000);
    assert.equal(isJwtExpired(validToken), false);
  });

  it('is expired a second after exp', (t) => {
    t.mock.method(Date, 'now', () => 1579986983 * 1000);
    assert.equal(isJwtExpired(validToken), true);
  });
});
