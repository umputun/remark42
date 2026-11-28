import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { createIntl } from '../common/intl-message.ts';

import {
  asApiError,
  backendErrorCode,
  errorMessages,
  extractErrorMessageFromResponse,
  RequestError,
} from './errorUtils.ts';

/**
 * This is what a reader is shown when something fails, so a wrong answer here is the failure they
 * see instead of the real one. It runs against the real catalogue through the real formatter, both
 * of which the dependency-free layer builds for itself.
 */
const intl = createIntl('en', {});

describe('extractErrorMessageFromResponse', () => {
  it('passes a plain string through as the message', () => {
    assert.equal(extractErrorMessageFromResponse('something the caller wrote', intl), 'something the caller wrote');
  });

  it('renders a request error from its own code', () => {
    assert.equal(
      extractErrorMessageFromResponse(new RequestError('x', 403), intl),
      "You don't have permission to do this."
    );
  });

  it('renders the connection failure the fetcher raises when the request never left', () => {
    assert.equal(
      extractErrorMessageFromResponse(new RequestError('x', 'fetch-error'), intl),
      'Failed to fetch. Please check your internet connection or try again a bit later'
    );
  });

  it('falls back for anything it does not recognize', () => {
    const fallback = 'Something went wrong. Please try again a bit later.';

    assert.equal(extractErrorMessageFromResponse(null, intl), fallback);
    assert.equal(extractErrorMessageFromResponse(undefined, intl), fallback);
    assert.equal(extractErrorMessageFromResponse(42, intl), fallback);
  });

  it('renders an error the backend sent as json from its own code', () => {
    assert.equal(
      extractErrorMessageFromResponse({ code: 3, error: 'no permission' }, intl),
      "You don't have permission for this operation."
    );
  });

  it('falls back for json whose code has no message of its own', () => {
    const fallback = 'Something went wrong. Please try again a bit later.';

    assert.equal(extractErrorMessageFromResponse({ code: 0, error: 'internal' }, intl), fallback);
    assert.equal(extractErrorMessageFromResponse({ code: 403, error: 'a status, not a code' }, intl), fallback);
    assert.equal(extractErrorMessageFromResponse({ error: 'no code at all' }, intl), fallback);
  });

  it('renders in the reader locale when the catalogue carries a translation', () => {
    const ru = createIntl('ru', { 'errors.forbidden': 'Доступ запрещён.' });

    assert.equal(extractErrorMessageFromResponse(new RequestError('x', 403), ru), 'Доступ запрещён.');
  });

  // this is the error path, so a throw here would replace whatever went wrong with nothing at all
  // for the reader. no construction site passes such a code today, and nothing but this would
  // catch a fourth one that did
  it('falls back for a code the catalogue does not carry', () => {
    const fallback = 'Something went wrong. Please try again a bit later.';

    assert.equal(extractErrorMessageFromResponse(new RequestError('x', 9999), intl), fallback);
    assert.equal(extractErrorMessageFromResponse(new RequestError('x', 'made-up'), intl), fallback);
  });
});

describe('backendErrorCode', () => {
  it('returns every code the backend defines a message for', () => {
    for (let code = 1; code <= 20; code++) {
      assert.equal(backendErrorCode({ code, error: 'x' }), code);
    }
  });

  // code 0 is ErrInternal, sent for many unrelated failures, so it must not replace the message
  // the HTTP status picks
  it('leaves out code 0', () => {
    assert.equal(backendErrorCode({ code: 0, error: 'x' }), undefined);
  });

  // the catalogue has entries keyed by HTTP status too, which are not the backend's codes
  it('does not take an HTTP status or an unknown number for a code', () => {
    for (const code of [401, 403, 409, 429, 500, 21, 9999, -1]) {
      assert.equal(backendErrorCode({ code }), undefined, `${code} was taken for a code`);
    }
  });

  it('returns undefined for anything that is not an error body with a numeric code', () => {
    for (const body of [undefined, null, 'error', 3, [], {}, { code: '3' }, { code: 'fetch-error' }]) {
      assert.equal(backendErrorCode(body), undefined, `${JSON.stringify(body)} gave a code`);
    }
  });
});

describe('asApiError', () => {
  it("returns a body in the shape of the backend's error response", () => {
    const body = { code: 17, details: "can't process token", error: 'token is expired' };

    assert.equal(asApiError(body), body);
    assert.deepEqual(asApiError({ error: 'no code' }), { error: 'no code' });
  });

  it('returns undefined for anything without an error text', () => {
    for (const body of [undefined, null, 'error', 3, [], {}, { code: 3 }, { error: 3 }]) {
      assert.equal(asApiError(body), undefined, `${JSON.stringify(body)} was taken for an error body`);
    }
  });
});

describe('RequestError', () => {
  it('is an Error, so it survives a throw and carries a stack', () => {
    const err = new RequestError('gone wrong', 404);

    assert.ok(err instanceof Error);
    assert.equal(err.message, 'gone wrong');
  });

  it('keeps the message under `error` as well, which the panels read', () => {
    assert.equal(new RequestError('gone wrong', 404).error, 'gone wrong');
  });

  it('keeps a string code as a string instead of coercing it', () => {
    assert.equal(new RequestError('x', 'fetch-error').code, 'fetch-error');
  });

  it('keeps the HTTP status apart from the code the message is looked up by', () => {
    const err = new RequestError('x', 17, 409);

    assert.equal(err.code, 17);
    assert.equal(err.status, 409);
    assert.equal(new RequestError('x', 'fetch-error').status, undefined);
  });
});

/**
 * The catalogue is indexed by three keys directly in code paths that have no fallback: 0 by the
 * extractor above, 500 by the fetcher when a body will not parse, and fetch-error when the request
 * never left. Losing any of them turns an error into a crash.
 */
describe('errorMessages', () => {
  it('carries the keys the code indexes without a guard', () => {
    for (const key of [0, 500, 'fetch-error'] as const) {
      assert.ok(errorMessages[key], `errorMessages[${String(key)}] is missing`);
    }
  });

  it('gives every entry an id and a message', () => {
    for (const [key, descriptor] of Object.entries(errorMessages)) {
      assert.ok(descriptor.id, `${key} has no id`);
      assert.ok(descriptor.defaultMessage, `${key} has no defaultMessage`);
    }
  });

  // a duplicated id makes two errors render as one message, and nothing else reports it: extraction
  // succeeds, translation succeeds, and the wrong text simply appears
  it('gives every entry an id of its own', () => {
    const ids = Object.values(errorMessages).map((d) => d.id);

    assert.equal(new Set(ids).size, ids.length);
  });

  it('renders every entry through the formatter without falling back to its id', () => {
    for (const [key, descriptor] of Object.entries(errorMessages)) {
      const rendered = intl.formatMessage(descriptor);

      assert.notEqual(rendered, descriptor.id, `${key} rendered as its own id`);
      assert.ok(rendered.length > 0, `${key} rendered empty`);
    }
  });
});
