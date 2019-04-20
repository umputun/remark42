/*
 * connected comment is not exported in index.ts to avoid leaking redux import into last-comments
 * and should be importded explicitly
 */

import { Comment as CommentType, User, BlockTTL, CommentMode } from '@app/common/types';

import { connect } from 'preact-redux';

import { StoreState, StoreDispatch } from '@app/store';
import {
  addComment,
  removeComment,
  updateComment,
  setPinState,
  putVote,
  setCommentMode,
} from '@app/store/comments/actions';
import { setCollapse } from '@app/store/thread/actions';
import { blockUser, unblockUser, setVirifiedStatus } from '@app/store/user/actions';

import { Comment, Props } from './comment';
import { getCommentMode } from '@app/store/comments/getters';
import { uploadImage } from '@app/common/api';
import { getThreadIsCollapsed } from '@app/store/thread/getters';

const mapProps = (state: StoreState, cprops: { data: CommentType }) => {
  const props: Pick<
    Props,
    'editMode' | 'user' | 'isUserBanned' | 'post_info' | 'isCommentsDisabled' | 'theme' | 'collapsed'
  > = {
    editMode: getCommentMode(state, cprops.data.id),
    user: state.user,
    isUserBanned: state.bannedUsers.find(u => u.id === cprops.data.user.id) !== undefined,
    post_info: state.info,
    isCommentsDisabled: state.info.read_only || false,
    theme: state.theme,
    collapsed: getThreadIsCollapsed(state, cprops.data),
  };
  return props;
};

const mapDispatchToProps = (dispatch: StoreDispatch) => {
  const props: Pick<
    Props,
    | 'addComment'
    | 'updateComment'
    | 'removeComment'
    | 'setReplyEditState'
    | 'collapseToggle'
    | 'setPinState'
    | 'putCommentVote'
    | 'blockUser'
    | 'unblockUser'
    | 'setVerifyStatus'
    | 'uploadImage'
  > = {
    addComment: (text: string, title: string, pid?: CommentType['id']) => dispatch(addComment(text, title, pid)),
    updateComment: (id: CommentType['id'], text: string) => dispatch(updateComment(id, text)),
    removeComment: (id: CommentType['id']) => dispatch(removeComment(id)),
    setReplyEditState: (id: CommentType['id'], mode: CommentMode) => dispatch(setCommentMode({ id, state: mode })),
    collapseToggle: (id: CommentType['id']) => dispatch(setCollapse(id)),
    setPinState: (id: CommentType['id'], value: boolean) => dispatch(setPinState(id, value)),
    putCommentVote: (id: CommentType['id'], value: number) => dispatch(putVote(id, value)),

    blockUser: (id: User['id'], name: User['name'], ttl: BlockTTL) => dispatch(blockUser(id, name, ttl)),
    unblockUser: (id: User['id']) => dispatch(unblockUser(id)),
    setVerifyStatus: (id: User['id'], value: boolean) => dispatch(setVirifiedStatus(id, value)),
    // should i made it as store action?
    uploadImage: (image: File) => uploadImage(image),
  };

  return props;
};

/** Comment component connected to redux */
export const ConnectedComment = connect(
  mapProps,
  mapDispatchToProps
)(Comment);
