import type { ComponentType } from 'preact';
import { h } from 'preact';
import { useLayoutEffect, useRef } from 'preact/hooks';

import type { IntlShape } from 'common/intl';
import { useIntl } from 'common/intl';
import type { Theme, Comment as CommentType, CommentMode, PostInfo, User } from 'common/types';
import type { getPreview, uploadImage } from 'common/api';
import { putCommentVote } from 'common/api';
import { StaticStore } from 'common/static-store';
import type { StoreState } from 'store';
import { useStore } from 'store/context';
import { patchComment } from 'store/comments/actions';
import { signInDeps } from 'components/auth/auth';
import type { Props as CommentFormProps } from 'components/comment-form';
import { CommentForm as CommentFormWrapper } from 'components/comment-form';
import { loadEmoji } from 'components/comment-form/emoji-data';
import { subscribeByEmailArgs } from 'components/comment-form/__subscribe-by-email/comment-form__subscribe-by-email';
import { subscribeByTelegramArgs } from 'components/comment-form/__subscribe-by-telegram/comment-form__subscribe-by-telegram';

import type { boundActions } from './connected-comment';
import { mountComment } from './comment-element.js';

import './styles';

export type CommentProps = {
  user: User | null;
  /** given, the comment draws its reply and edit forms */
  CommentForm?: ComponentType<CommentFormProps>;
  data: CommentType;
  repliesCount?: number;
  post_info?: PostInfo;
  /** whether comment's user is banned */
  isUserBanned?: boolean;
  isCommentsDisabled: boolean;
  /** edit mode: is comment should have reply, or edit Input */
  editMode?: CommentMode;
  /**
   * "main" view used in main case,
   * "pinned" view used in pinned block,
   * "user" is for user comments widget
   */
  view: 'main' | 'pinned' | 'user';
  /** defines whether comment should have reply/edit actions */
  disabled?: boolean;
  collapsed?: boolean;
  theme: Theme;
  inView?: boolean;
  level?: number;
  mix?: string;
  getPreview?: typeof getPreview;
  uploadImage?: typeof uploadImage;
  intl: IntlShape;
} & Partial<typeof boundActions>;

type Deps = Parameters<typeof mountComment>[2];

/**
 * Mounts the plain comment into an article preact renders without props or children, so a
 * re-render of the thread leaves the article's attributes and content to the plain component.
 */
export function Comment(props: CommentProps) {
  const contextIntl = useIntl();
  const store = useStore<StoreState, never>();
  const root = useRef<HTMLElement>(null);
  const comment = useRef<ReturnType<typeof mountComment> | null>(null);

  const commentProps: Parameters<typeof mountComment>[1] = {
    data: props.data,
    user: props.user,
    repliesCount: props.repliesCount,
    post_info: props.post_info,
    isUserBanned: props.isUserBanned,
    editMode: props.editMode,
    view: props.view,
    disabled: props.disabled,
    collapsed: props.collapsed,
    theme: props.theme,
    inView: props.inView,
    level: props.level,
    mix: props.mix,
  };
  const deps: Deps = {
    intl: props.intl,
    partsIntl: contextIntl,
    votes: {
      intl: contextIntl,
      putVote: (id, vote) => putCommentVote({ id, vote }),
      patch: (patch) => store.dispatch(patchComment(patch) as never),
      positiveScore: () => Boolean(StaticStore.config.positive_score),
    },
    actions: {
      addComment: props.addComment,
      updateComment: props.updateComment,
      removeComment: props.removeComment,
      setReplyEditState: props.setReplyEditState,
      setPinState: props.setPinState,
      blockUser: props.blockUser,
      unblockUser: props.unblockUser,
      hideUser: props.hideUser,
      setVerifiedStatus: props.setVerifiedStatus,
    },
    getPreview: props.getPreview,
    uploadImage: props.uploadImage,
    // read from the store as the comment draws, without a subscription of its own in every
    // comment: the connected comment draws again whenever the reader or the theme changes
    form: props.CommentForm && formBase(store.getState(), store.dispatch as never, contextIntl),
  };

  useLayoutEffect(() => {
    comment.current = mountComment(root.current!, commentProps, deps);
    return () => comment.current?.destroy();
    // mounted once; the props and deps of every later render reach it through update below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const mounted = useRef(false);
  useLayoutEffect(() => {
    // the first render is the mount above; each one after it is a render of the thread
    if (mounted.current) {
      comment.current?.update(commentProps, deps);
    }
    mounted.current = true;
  });

  return <article ref={root} />;
}

/** what the forms under a comment take from the store and the locale, as the preact form took it */
function formBase(
  state: StoreState,
  dispatch: Parameters<typeof subscribeByEmailArgs>[3] & Parameters<typeof signInDeps>[2],
  intl: IntlShape
): NonNullable<Deps['form']> {
  const { user, theme } = state;
  return {
    nextTextareaId: () => {
      CommentFormWrapper.textareaCounter += 1;
      return `textarea_${CommentFormWrapper.textareaCounter}`;
    },
    loadEmoji: () => loadEmoji(),
    theme,
    signIn: signInDeps(intl, theme, dispatch),
    rssIntl: intl,
    email: subscribeByEmailArgs(user, theme, intl, dispatch),
    telegram: subscribeByTelegramArgs(user, theme, intl),
  };
}
