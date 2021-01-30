import { IntlShape, defineMessages, MessageDescriptor } from 'react-intl';

const messages = defineMessages({
  failedFetch: {
    id: 'errors.failed-fetch',
    defaultMessage: 'Failed to fetch. Please check your internet connection or try again a bit later',
    description: {
      code: -2,
    },
  },
  0: {
    id: 'errors.0',
    defaultMessage: 'Something went wrong. Please try again a bit later.',
    description: {
      code: 0,
    },
  },
  1: {
    id: 'errors.1',
    defaultMessage: 'Comment cannot be found. Please refresh the page and try again.',
    description: {
      code: 1,
    },
  },
  2: {
    id: 'errors.2',
    defaultMessage: 'Failed to unmarshal incoming request.',
    description: {
      code: 2,
    },
  },
  3: {
    id: 'errors.3',
    defaultMessage: `You don't have permission for this operation.`,
    description: {
      code: 3,
    },
  },
  4: {
    id: 'errors.4',
    defaultMessage: `Invalid comment data.`,
    description: {
      code: 4,
    },
  },
  5: {
    id: 'errors.5',
    defaultMessage: `Comment cannot be found.  Please refresh the page and try again.`,
    description: {
      code: 5,
    },
  },
  6: {
    id: 'errors.6',
    defaultMessage: `Site cannot be found.  Please refresh the page and try again.`,
    description: {
      code: 6,
    },
  },
  7: {
    id: 'errors.7',
    defaultMessage: `User has been blocked.`,
    description: {
      code: 7,
    },
  },
  8: {
    id: 'errors.8',
    defaultMessage: `User has been blocked.`,
    description: {
      code: 8,
    },
  },
  9: {
    id: 'errors.9',
    defaultMessage: `Comment changing failed. Please try again a bit later.`,
    description: {
      code: 9,
    },
  },
  10: {
    id: 'errors.10',
    defaultMessage: `It is too late to edit the comment.`,
    description: {
      code: 10,
    },
  },
  11: {
    id: 'errors.11',
    defaultMessage: `Comment already has reply, editing is not possible.`,
    description: {
      code: 11,
    },
  },
  12: {
    id: 'errors.12',
    defaultMessage: `Cannot save voting result. Please try again a bit later.`,
    description: {
      code: 12,
    },
  },
  13: {
    id: 'errors.13',
    defaultMessage: `You cannot vote for your own comment.`,
    description: {
      code: 13,
    },
  },
  14: {
    id: 'errors.14',
    defaultMessage: `You have already voted for the comment.`,
    description: {
      code: 14,
    },
  },
  15: {
    id: 'errors.15',
    defaultMessage: `Too many votes for the comment.`,
    description: {
      code: 15,
    },
  },
  16: {
    id: 'errors.16',
    defaultMessage: `Min score reached for the comment.`,
    description: {
      code: 16,
    },
  },
  17: {
    id: 'errors.17',
    defaultMessage: `Action rejected. Please try again a bit later.`,
    description: {
      code: 17,
    },
  },
  18: {
    id: 'errors.18',
    defaultMessage: `Requested file cannot be found.`,
    description: {
      code: 18,
    },
  },
});

/**
 * map of codes that server returns in its response in case of error
 * to client readable version
 */
const errorMessageForCodes = new Map<number, MessageDescriptor>();

Object.entries(messages).forEach(([, messageDescriptor]) => {
  errorMessageForCodes.set(messageDescriptor.description.code, messageDescriptor);
});

export const httpMessages = defineMessages({
  notAuthorized: {
    id: 'errors.not-authorized',
    defaultMessage: 'Not authorized.',
  },
  forbidden: {
    id: 'errors.forbidden',
    defaultMessage: 'Forbidden.',
  },
  toManyRequest: {
    id: 'errors.to-many-request',
    defaultMessage: 'You have reached maximum request limit.',
  },
  unexpectedError: {
    id: 'errors.unexpected-error',
    defaultMessage: 'Something went wrong.',
  },
});

/**
 * map of http rest codes to ui label, used by fetcher to generate error with `-1` code
 */
export const httpErrorMap = new Map([
  [401, httpMessages.notAuthorized],
  [403, httpMessages.forbidden],
  [429, httpMessages.toManyRequest],
]);

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
  const defaultErrorMessage = intl.formatMessage(errorMessageForCodes.get(0) || messages['0']);
  if (!response) {
    return defaultErrorMessage;
  }

  if (typeof response === 'string') {
    return response;
  }

  if (
    typeof response.code === 'number' &&
    (errorMessageForCodes.has(response.code) || httpErrorMap.has(response.code))
  ) {
    const messageDescriptor =
      errorMessageForCodes.get(response.code) || httpErrorMap.get(response.code) || messages['0'];
    return intl.formatMessage(messageDescriptor);
  }

  return defaultErrorMessage;
}

export class RequestError extends Error {
  code: number;
  error: string;

  constructor(message: string, code: number) {
    super(message);

    this.code = code;
    this.error = message;
  }
}
