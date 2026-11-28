import type { IntlShape } from '../common/intl-message.ts';
import { defineMessages } from '../common/intl-message.ts';
import type { ApiError } from '../common/types';

export const errorMessages = defineMessages<string | number>({
  'fetch-error': {
    id: 'errors.failed-fetch',
    defaultMessage: 'Failed to fetch. Please check your internet connection or try again a bit later',
  },
  0: {
    id: 'errors.0',
    defaultMessage: 'Something went wrong. Please try again a bit later.',
  },
  1: {
    id: 'errors.1',
    defaultMessage: 'Comment cannot be found. Please refresh the page and try again.',
  },
  2: {
    id: 'errors.2',
    defaultMessage: 'The request could not be read. Please refresh the page and try again.',
  },
  3: {
    id: 'errors.3',
    defaultMessage: `You don't have permission for this operation.`,
  },
  4: {
    id: 'errors.4',
    defaultMessage: 'Invalid comment data.',
  },
  5: {
    id: 'errors.5',
    defaultMessage: 'Comments for this page could not be loaded. Please refresh the page and try again.',
  },
  6: {
    id: 'errors.6',
    defaultMessage: 'Site cannot be found. Please refresh the page and try again.',
  },
  7: {
    id: 'errors.7',
    defaultMessage: 'User has been blocked.',
  },
  8: {
    id: 'errors.8',
    defaultMessage: 'Comments on this page are read-only.',
  },
  9: {
    id: 'errors.9',
    defaultMessage: 'Comment changing failed. Please try again a bit later.',
  },
  10: {
    id: 'errors.10',
    defaultMessage: 'It is too late to edit the comment.',
  },
  11: {
    id: 'errors.11',
    defaultMessage: 'The comment already has a reply, so it can no longer be edited.',
  },
  12: {
    id: 'errors.12',
    defaultMessage: 'Voting failed. Please try again in a little while.',
  },
  13: {
    id: 'errors.13',
    defaultMessage: 'You cannot vote for your own comment.',
  },
  14: {
    id: 'errors.14',
    defaultMessage: 'You have already voted for the comment.',
  },
  15: {
    id: 'errors.15',
    defaultMessage: 'This comment has reached the maximum number of votes.',
  },
  16: {
    id: 'errors.16',
    defaultMessage: 'This comment has reached the minimum score.',
  },
  17: {
    id: 'errors.17',
    defaultMessage: 'Action rejected. Please try again a bit later.',
  },
  18: {
    id: 'errors.18',
    defaultMessage: 'Requested file cannot be found.',
  },
  19: {
    id: 'errors.19',
    defaultMessage: 'Comment contains restricted words.',
  },
  20: {
    id: 'errors.20',
    defaultMessage: 'Uploaded image not found. Please try to upload it again.',
  },
  401: {
    id: 'errors.not-authorized',
    defaultMessage: 'Please sign in to do this.',
  },
  403: {
    id: 'errors.forbidden',
    defaultMessage: "You don't have permission to do this.",
  },
  409: {
    id: 'errors.conflict',
    defaultMessage: 'This action cannot be completed right now. Please refresh the page and try again.',
  },
  429: {
    id: 'errors.to-many-request',
    defaultMessage: 'Too many requests. Please wait a moment and try again.',
  },
  500: {
    id: 'errors.unexpected-error',
    defaultMessage: 'Something went wrong.',
  },
});

export type FetcherError = string | ApiError | RequestError | unknown;

/**
 * Returns the backend's error code from a JSON error body, when the catalogue has a message of its
 * own for it, and undefined otherwise.
 *
 * Code 0 is `ErrInternal`, which the backend sends for many unrelated failures, so it is left out:
 * letting it win would replace a specific message picked by the HTTP status with the generic one.
 * The catalogue also has entries keyed by HTTP status, so a code only counts when its entry is one
 * of the backend's own `errors.<code>` messages.
 */
export function backendErrorCode(body: unknown): number | undefined {
  if (typeof body !== 'object' || body === null) {
    return undefined;
  }

  const { code } = body as { code?: unknown };

  if (typeof code !== 'number' || code === 0 || errorMessages[code]?.id !== `errors.${code}`) {
    return undefined;
  }

  return code;
}

export function extractErrorMessageFromResponse(response: FetcherError, intl: IntlShape): string {
  if (typeof response === 'string') {
    return response;
  }

  // a code with no entry falls back instead of indexing to undefined: this is the error path, so
  // a throw here replaces whatever went wrong with nothing at all for the reader
  if (response instanceof RequestError) {
    return intl.formatMessage(errorMessages[response.code] ?? errorMessages[0]);
  }

  return intl.formatMessage(errorMessages[backendErrorCode(response) ?? 0]);
}

/** returns the body if it has the shape of the backend's error response, undefined otherwise */
export function asApiError(body: unknown): ApiError | undefined {
  if (typeof body !== 'object' || body === null || typeof (body as { error?: unknown }).error !== 'string') {
    return undefined;
  }

  return body as ApiError;
}

export class RequestError extends Error {
  /** the catalogue key the message is looked up by: a backend error code, an HTTP status or `fetch-error` */
  code: number | string;
  error: string;
  /** the HTTP status of the response, absent when there was no response */
  status?: number;
  /** the error body the backend sent, absent when it sent none */
  body?: ApiError;

  constructor(message: string, code: number | string, status?: number, body?: ApiError) {
    super(message);

    this.code = code;
    this.error = message;
    this.status = status;
    this.body = body;
  }
}
