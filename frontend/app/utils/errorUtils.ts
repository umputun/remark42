/**
 * map of codes that server returns in its response in case of error
 * to client readable version
 */
const errorMessageForCodes = new Map([
  [-2, 'Failed to fetch. Please check your internet connection.'],
  [0, 'Something went wrong. Please try again a bit later.'],
  [1, 'Comment cannot be found. Please refresh the page and try again.'],
  [2, 'Failed to unmarshal incoming request.'],
  [3, "You don't have permission for this operaton."],
  [4, 'Invalid comment data.'],
  [5, 'Comment cannot be found.  Please refresh the page and try again.'],
  [6, 'Site cannot be found.  Please refresh the page and try again.'],
  [7, 'User has been blocked.'],
  [8, 'This post is read only.'],
  [9, 'Comment changing failed. Please try again a bit later.'],
  [10, 'It is too late to edit the comment.'],
  [11, 'Comment already has reply, editing is not possible.'],
  [12, 'Cannot save voting result. Please try again a bit later.'],
  [13, 'You cannot vote for your own comment.'],
  [14, 'You have already voted for the comment.'],
  [15, 'Too many votes for the comment.'],
  [16, 'Min score reached for the comment.'],
  [17, 'Action rejected. Please try again a bit later.'],
  [18, 'Requested file cannot be found.'],
]);

/**
 * map of http rest codes to ui label, used by fetcher to generate error with `-1` code
 */
export const httpErrorMap = new Map([
  [401, 'Not authorized.'],
  [403, 'Forbidden.'],
  [429, 'You have reached maximum request limit.'],
]);

export function isFailedFetch(e?: Error): boolean {
  return Boolean(e && e.message && e.message === `Failed to fetch`);
}

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

export function extractErrorMessageFromResponse(response: FetcherError): string {
  const defaultErrorMessage = errorMessageForCodes.get(0) as string;

  if (!response) {
    return defaultErrorMessage;
  }

  if (typeof response === 'string') {
    return response;
  }

  if (response.code === -1) {
    return response.error;
  }

  if (typeof response.details === 'string') {
    return response.details.charAt(0).toUpperCase() + response.details.substring(1);
  }

  if (typeof response.code === 'number' && errorMessageForCodes.has(response.code)) {
    return errorMessageForCodes.get(response.code)!;
  }

  return defaultErrorMessage;
}
