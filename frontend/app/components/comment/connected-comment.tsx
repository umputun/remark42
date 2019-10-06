/*
 * connected comment is not exported in index.ts to avoid leaking redux import into last-comments
 * and should be importded explicitly
 */

/** @jsx createElement */

import './styles';

import { createElement, FunctionComponent } from 'preact';

import { Comment as CommentType } from '@app/common/types';

import { useStore } from 'react-redux';

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
import { useActions } from '@app/hooks/useAction';

type ProvidedProps = Pick<
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
>;

const mapStateToProps = (state: StoreState, cprops: { data: CommentType }) => {
  const props: ProvidedProps = {
    editMode: getCommentMode(cprops.data.id)(state),
    user: state.user,
    isUserBanned: cprops.data.user.block || state.bannedUsers.find(u => u.id === cprops.data.user.id) !== undefined,
    post_info: state.info,
    isCommentsDisabled: state.info.read_only || false,
    theme: state.theme,
    collapsed: getThreadIsCollapsed(cprops.data)(state),
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

export const ConnectedComment: FunctionComponent<Omit<Props, keyof (ProvidedProps & typeof bindActions)>> = props => {
  const providedProps = mapStateToProps(useStore().getState(), props);
  const actions = useActions(boundActions);
  return <Comment {...props} {...providedProps} {...actions} />;
};
