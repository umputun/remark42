import { IntlShape, defineMessages } from 'react-intl';

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
    defaultMessage: 'Failed to unmarshal incoming request.',
  },
  3: {
    id: 'errors.3',
    defaultMessage: `You don't have permission for this operation.`,
  },
  4: {
    id: 'errors.4',
    defaultMessage: `Invalid comment data.`,
  },
  5: {
    id: 'errors.5',
    defaultMessage: `Comment cannot be found. Please refresh the page and try again.`,
  },
  6: {
    id: 'errors.6',
    defaultMessage: `Site cannot be found. Please refresh the page and try again.`,
  },
  7: {
    id: 'errors.7',
    defaultMessage: `User has been blocked.`,
  },
  8: {
    id: 'errors.8',
    defaultMessage: `User has been blocked.`,
  },
  9: {
    id: 'errors.9',
    defaultMessage: `Comment changing failed. Please try again a bit later.`,
  },
  10: {
    id: 'errors.10',
    defaultMessage: `It is too late to edit the comment.`,
  },
  11: {
    id: 'errors.11',
    defaultMessage: `Comment already has reply, editing is not possible.`,
  },
  12: {
    id: 'errors.12',
    defaultMessage: `Cannot save voting result. Please try again a bit later.`,
  },
  13: {
    id: 'errors.13',
    defaultMessage: `You cannot vote for your own comment.`,
  },
  14: {
    id: 'errors.14',
    defaultMessage: `You have already voted for the comment.`,
  },
  15: {
    id: 'errors.15',
    defaultMessage: `Too many votes for the comment.`,
  },
  16: {
    id: 'errors.16',
    defaultMessage: `Min score reached for the comment.`,
  },
  17: {
    id: 'errors.17',
    defaultMessage: `Action rejected. Please try again a bit later.`,
  },
  18: {
    id: 'errors.18',
    defaultMessage: `Requested file cannot be found.`,
  },
  19: {
    id: 'errors.19',
    defaultMessage: 'Comment contains restricted words.',
  },
  401: {
    id: 'errors.not-authorized',
    defaultMessage: 'Not authorized.',
  },
  403: {
    id: 'errors.forbidden',
    defaultMessage: 'Forbidden.',
  },
  429: {
    id: 'errors.to-many-request',
    defaultMessage: 'You have reached maximum request limit.',
  },
  500: {
    id: 'errors.unexpected-error',
    defaultMessage: 'Something went wrong.',
  },
});

export type FetcherError =
  | string
  | {
      /**
       * Error code, that is part of server error response.
       *
       * Note that -1 is reserved for error where `error` field shall be used directly
       */
      code?: number;
      details?: string;
      error: string;
    };

export function extractErrorMessageFromResponse(response: FetcherError, intl: IntlShape): string {
  if (typeof response === 'string') {
    return response;
  }

  if (typeof response.code === 'number' && errorMessages[response.code]) {
    return intl.formatMessage(errorMessages[response.code]);
  }

  return intl.formatMessage(errorMessages[0]);
}

export class RequestError extends Error {
  code: number | string;
  error: string;

  constructor(message: string, code: number | string) {
    super(message);

    this.code = code;
    this.error = message;
  }
}
