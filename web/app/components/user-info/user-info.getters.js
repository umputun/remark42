export const getUserComments = (state, userId) => state.userComments[userId] || null;
export const getIsLoadingUserComments = (state, userId) => state.isLoadingUserComments[userId] || false;
