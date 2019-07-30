/*
 * connected comment is not exported in index.ts to avoid leaking redux import into last-comments
 * and should be importded explicitly
 */

import './styles';

import { Comment as CommentType } from '@app/common/types';

import { connect } from 'preact-redux';

import { StoreState } from '@app/store';
import {
  addComment,
  removeComment,
  updateComment,
  setPinState,
  putVote,
  setCommentMode,
} from '@app/store/comments/actions';
import { setCollapse } from '@app/store/thread/actions';
import { blockUser, unblockUser, hideUser, setVerifiedStatus } from '@app/store/user/actions';

import { Comment, Props } from './comment';
import { getCommentMode } from '@app/store/comments/getters';
import { uploadImage, getPreview } from '@app/common/api';
import { getThreadIsCollapsed } from '@app/store/thread/getters';
import { bindActions } from '@app/utils/actionBinder';

const mapStateToProps = (state: StoreState, cprops: { data: CommentType }) => {
  const props: Pick<
    Props,
    | 'editMode'
    | 'user'
    | 'isUserBanned'
    | 'post_info'
    | 'isCommentsDisabled'
    | 'theme'
    | 'collapsed'
    | 'getPreview'
    | 'uploadImage'
  > = {
    editMode: getCommentMode(state, cprops.data.id),
    user: state.user,
    isUserBanned: cprops.data.user.block || state.bannedUsers.find(u => u.id === cprops.data.user.id) !== undefined,
    post_info: state.info,
    isCommentsDisabled: state.info.read_only || false,
    theme: state.theme,
    collapsed: getThreadIsCollapsed(state, cprops.data),
    getPreview,
    uploadImage,
  };
  return props;
};

export const boundActions = bindActions({
  addComment,
  updateComment,
  removeComment,
  setReplyEditState: setCommentMode,
  setCollapse,
  setPinState,
  putCommentVote: putVote,
  blockUser,
  unblockUser,
  hideUser,
  setVerifyStatus: setVerifiedStatus,
});

/** Comment component connected to redux */
export const ConnectedComment = connect(
  mapStateToProps,
  boundActions as Partial<typeof boundActions>
)(Comment);
