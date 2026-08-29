import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { createIntl } from '../common/intl-message.ts';

import { errorMessages, extractErrorMessageFromResponse, RequestError } from './errorUtils.ts';

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
    assert.equal(extractErrorMessageFromResponse(new RequestError('x', 403), intl), 'Forbidden.');
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

  // an error the backend sent as json is not a RequestError, so its own code is never looked up
  // and it reads as the generic failure. worth stating, because the shape suggests otherwise
  it('does not read the code off a plain error object', () => {
    assert.equal(
      extractErrorMessageFromResponse({ code: 3, error: 'no permission' }, intl),
      'Something went wrong. Please try again a bit later.'
    );
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
