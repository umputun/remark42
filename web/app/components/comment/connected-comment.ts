/*
 * connected comment is not exported in index.ts to avoid leaking redux import into last-comments
 * and should be importded explicitly
 */

import { Comment as CommentType, User, BlockTTL } from '@app/common/types';

import { connect } from 'preact-redux';
import { Derequire } from '@app/utils/derequire';

import { StoreState, StoreDispatch } from '@app/store';
import { addComment, removeComment, updateComment, setPinState, putVote } from '@app/store/comments/actions';
import { setCollapse } from '@app/store/thread/actions';
import { blockUser, unblockUser, setVirifiedStatus } from '@app/store/user/actions';

import { Comment } from './comment';

const mapProps = (state: StoreState, cprops: { data: CommentType }) => {
  const props = {
    user: state.user,
    isUserBanned: state.bannedUsers.find(u => u.id === cprops.data.user.id) !== undefined,
    post_info: state.info,
    isCommentsDisabled: state.info.read_only || false,
    theme: state.theme,
    collapsed: state.collapsedThreads[cprops.data.id] === true,
  };
  return props as Derequire<typeof props, 'isUserBanned' | 'collapsed'>;
};

const mapDispatchToProps = (dispatch: StoreDispatch) => {
  const props = {
    addComment: (text: string, title: string, pid?: CommentType['id']) => dispatch(addComment(text, title, pid)),
    updateComment: (id: CommentType['id'], text: string) => dispatch(updateComment(id, text)),
    removeComment: (id: CommentType['id']) => dispatch(removeComment(id)),
    collapseToggle: (id: CommentType['id']) => dispatch(setCollapse(id)),
    setPinState: (id: CommentType['id'], value: boolean) => dispatch(setPinState(id, value)),
    putCommentVote: (id: CommentType['id'], value: number) => dispatch(putVote(id, value)),

    blockUser: (id: User['id'], name: User['name'], ttl: BlockTTL) => dispatch(blockUser(id, name, ttl)),
    unblockUser: (id: User['id']) => dispatch(unblockUser(id)),
    setVerifyStatus: (id: User['id'], value: boolean) => dispatch(setVirifiedStatus(id, value)),
  };

  // Making all optional to meet component expectations
  return props as Partial<typeof props>;
};

/** Comment component connected to redux */
export const ConnectedComment = connect(
  mapProps,
  mapDispatchToProps
)(Comment);
