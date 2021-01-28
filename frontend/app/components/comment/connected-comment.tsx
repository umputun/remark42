/*
 * connected comment is not exported in index.ts to avoid leaking redux import into last-comments
 * and should be importded explicitly
 */

import './styles';

import { h, FunctionComponent } from 'preact';

import { Comment as CommentType } from 'common/types';

import { useStore } from 'react-redux';

import { StoreState } from 'store';
import { addComment, removeComment, updateComment, setPinState, putVote, setCommentMode } from 'store/comments/actions';
import { blockUser, unblockUser, hideUser, setVerifiedStatus } from 'store/user/actions';

import Comment, { CommentProps } from './comment';
import { getCommentMode } from 'store/comments/getters';
import { uploadImage, getPreview } from 'common/api';
import { getThreadIsCollapsed } from 'store/thread/getters';
import { bindActions } from 'utils/actionBinder';
import { useActions } from 'hooks/useAction';
import { useIntl } from 'react-intl';

type ProvidedProps = Pick<
  CommentProps,
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
    isUserBanned: cprops.data.user.block || state.bannedUsers.find((u) => u.id === cprops.data.user.id) !== undefined,
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
  setPinState,
  putCommentVote: putVote,
  blockUser,
  unblockUser,
  hideUser,
  setVerifiedStatus,
});

export const ConnectedComment: FunctionComponent<Omit<CommentProps, keyof (ProvidedProps & typeof bindActions)>> = (
  props
) => {
  const providedProps = mapStateToProps(useStore().getState(), props);
  const actions = useActions(boundActions);
  const intl = useIntl();

  return <Comment {...props} {...providedProps} {...actions} intl={intl} />;
};
