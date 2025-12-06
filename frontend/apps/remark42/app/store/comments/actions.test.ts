import { mockStore } from '__stubs__/store';
import { LS_SORT_KEY } from 'common/constants';
import * as api from 'common/api';

import { updateSorting, setApprovalState } from './actions';
import { COMMENTS_SET_SORT, COMMENTS_EDIT } from './types';

describe('Store comments actions', () => {
  it('should save last sort to localstorage', async () => {
    const newSort = '+controversy';
    const store = mockStore({
      comments: {
        sort: '+active',
      },
      hiddenUsers: {},
    });

    await store.dispatch(updateSorting(newSort));

    const [setCommentAction] = store.getActions();

    expect(setCommentAction).toEqual({ type: COMMENTS_SET_SORT, payload: newSort });
    expect(localStorage.setItem).toHaveBeenCalledWith(LS_SORT_KEY, newSort);
  });
});

describe('setApprovalState', () => {
  const commentId = 'comment-123';
  const mockComment = {
    id: commentId,
    text: 'test comment',
    approved: false,
    user: {
      id: 'user-1',
      name: 'Test User',
    },
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should call approveComment API when value is true', async () => {
    const approveCommentSpy = jest.spyOn(api, 'approveComment').mockResolvedValue(undefined);
    const store = mockStore({
      comments: {
        allComments: {
          [commentId]: mockComment,
        },
      },
      hiddenUsers: {},
    });

    await store.dispatch(setApprovalState(commentId, true));

    expect(approveCommentSpy).toHaveBeenCalledWith(commentId);
    const actions = store.getActions();
    expect(actions[0].type).toBe(COMMENTS_EDIT);
    expect(actions[0].comment.approved).toBe(true);
  });

  it('should call disapproveComment API when value is false', async () => {
    const disapproveCommentSpy = jest.spyOn(api, 'disapproveComment').mockResolvedValue(undefined);
    const store = mockStore({
      comments: {
        allComments: {
          [commentId]: { ...mockComment, approved: true },
        },
      },
      hiddenUsers: {},
    });

    await store.dispatch(setApprovalState(commentId, false));

    expect(disapproveCommentSpy).toHaveBeenCalledWith(commentId);
    const actions = store.getActions();
    expect(actions[0].type).toBe(COMMENTS_EDIT);
    expect(actions[0].comment.approved).toBe(false);
  });

  it('should update the comment edit time', async () => {
    jest.spyOn(api, 'approveComment').mockResolvedValue(undefined);
    const store = mockStore({
      comments: {
        allComments: {
          [commentId]: mockComment,
        },
      },
      hiddenUsers: {},
    });

    await store.dispatch(setApprovalState(commentId, true));

    const actions = store.getActions();
    expect(actions[0].comment.edit).toBeDefined();
    expect(actions[0].comment.edit.time).toBeDefined();
  });
});
