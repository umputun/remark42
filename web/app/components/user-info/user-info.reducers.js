import { USER_INFO_FETCH_COMMENTS, USER_INFO_COMPLETE_FETCH_COMMENTS } from './user-info.actions';

export const userComments = (state = {}, action) => {
  switch (action.type) {
    case USER_INFO_FETCH_COMMENTS: {
      return {
        ...state,
        [action.userId]: [],
      };
    }
    case USER_INFO_COMPLETE_FETCH_COMMENTS:
      return {
        ...state,
        [action.userId]: action.comments,
      };
    default:
      return state;
  }
};

export const isLoadingUserComments = (state = {}, action) => {
  switch (action.type) {
    case USER_INFO_FETCH_COMMENTS: {
      return {
        ...state,
        [action.userId]: true,
      };
    }
    case USER_INFO_COMPLETE_FETCH_COMMENTS:
      return {
        ...state,
        [action.userId]: false,
      };
    default:
      return state;
  }
};
