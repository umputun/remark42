export const USER_INFO_FETCH_COMMENTS = 'USER_INFO/FETCH_COMMENTS';
export const fetchComments = userId => ({
  type: USER_INFO_FETCH_COMMENTS,
  userId,
});

export const USER_INFO_COMPLETE_FETCH_COMMENTS = 'USER_INFO/COMPLETE_FETCH_COMMENTS';
export const completeFetchComments = (userId, comments) => ({
  type: USER_INFO_COMPLETE_FETCH_COMMENTS,
  userId,
  comments,
});
