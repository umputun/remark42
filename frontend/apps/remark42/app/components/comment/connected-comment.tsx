/*
 * connected comment is not exported in index.ts to avoid leaking redux import into last-comments
 * and should be imported explicitly
 */

import './styles';

import { h, FunctionComponent } from 'preact';

import { useAppSelector } from 'store';
import { addComment, removeComment, updateComment, setPinState, setCommentMode } from 'store/comments/actions';
import { blockUser, unblockUser, hideUser, setVerifiedStatus } from 'store/user/actions';

import { Comment, CommentProps } from './comment';
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

export const boundActions = bindActions({
  addComment,
  updateComment,
  removeComment,
  setReplyEditState: setCommentMode,
  setPinState,
  blockUser,
  unblockUser,
  hideUser,
  setVerifiedStatus,
});

export const ConnectedComment: FunctionComponent<Omit<CommentProps, keyof (ProvidedProps & typeof bindActions)>> = (
  props
) => {
  const providedProps = useAppSelector((state): ProvidedProps => {
    return {
      editMode: getCommentMode(props.data.id)(state),
      user: state.user,
      isUserBanned: props.data.user.block || state.bannedUsers.find((u) => u.id === props.data.user.id) !== undefined,
      post_info: state.info,
      isCommentsDisabled: state.info.read_only || false,
      theme: state.theme,
      collapsed: getThreadIsCollapsed(props.data)(state),
      getPreview,
      uploadImage,
    };
  });
  const actions = useActions(boundActions);
  const intl = useIntl();

  return <Comment {...props} {...providedProps} {...actions} intl={intl} />;
};
