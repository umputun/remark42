const errorMessageForCodes = new Map([
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

export function extractErrorMessageFromResponse(response) {
  const defatulErrorMessage = 'Something went wrong. Please try again a bit later.';
  if (!(response && response.data)) {
    return defatulErrorMessage;
  }

  const responseData = response.data;

  if (typeof responseData.code === 'number' && errorMessageForCodes.has(responseData.code)) {
    return errorMessageForCodes.get(responseData.code);
  }

  if (typeof responseData.details === 'string') {
    return responseData.details;
  }

  if (typeof responseData === 'string') {
    return responseData;
  }

  return defatulErrorMessage;
}
