import { h } from 'preact';
import { useLayoutEffect, useRef, useState } from 'preact/hooks';

import type { IntlShape } from 'common/intl';
import { useIntl } from 'common/intl';
import type { User, Theme, Image } from 'common/types';
import { useTheme } from 'hooks/useTheme';
import { useSignInDeps } from 'components/auth/auth';

import { useSubscribeByEmailArgs } from './__subscribe-by-email/comment-form__subscribe-by-email';
import { useSubscribeByTelegramArgs } from './__subscribe-by-telegram/comment-form__subscribe-by-telegram';
import { mountCommentForm } from './comment-form-element.js';
import { loadEmoji } from './emoji-data';

export { messages } from './comment-form.messages';

export type Props = {
  id: string;
  user: User | null;
  errorMessage?: string;
  value?: string;
  mix?: string | string[];
  mode?: 'main' | 'edit' | 'reply';
  /**
   * Only meaningful when `mode` is `'edit'`: whether the comment being edited is itself a reply
   * (has its own parent), so its placeholder should match the reply form rather than the main one.
   */
  isReply?: boolean;
  theme: Theme;
  autofocus?: boolean;

  onSubmit(text: string, pageTitle: string): Promise<void>;
  getPreview(text: string): Promise<string>;
  /** action on cancel. optional as root input has no cancel option */
  onCancel?(): void;
  uploadImage?(image: File): Promise<Image>;
  intl: IntlShape;
};

/**
 * Mounts the plain comment form into a div preact renders without props or children, so a
 * re-render of the parent leaves the div's attributes and content to the plain component.
 */
export function CommentForm(props: Props) {
  // each form's field has an id of its own, taken once, when the form appears
  const [textareaId] = useState(() => {
    CommentForm.textareaCounter += 1;
    return `textarea_${CommentForm.textareaCounter}`;
  });
  const theme = useTheme();
  const contextIntl = useIntl();
  const signIn = useSignInDeps();
  const email = useSubscribeByEmailArgs();
  const telegram = useSubscribeByTelegramArgs();
  const root = useRef<HTMLDivElement>(null);
  const form = useRef<ReturnType<typeof mountCommentForm> | null>(null);

  const formProps: Parameters<typeof mountCommentForm>[1] = {
    id: props.id,
    textareaId,
    user: props.user,
    errorMessage: props.errorMessage,
    value: props.value,
    mix: props.mix,
    mode: props.mode,
    isReply: props.isReply,
    theme: props.theme,
    autofocus: props.autofocus,
  };
  const deps: Parameters<typeof mountCommentForm>[2] = {
    intl: props.intl,
    onSubmit: (text, pageTitle) => props.onSubmit(text, pageTitle),
    getPreview: (text) => props.getPreview(text),
    onCancel: props.onCancel && (() => props.onCancel?.()),
    uploadImage: props.uploadImage && ((image) => props.uploadImage!(image)),
    loadEmoji: () => loadEmoji(),
    theme,
    signIn,
    rss: { props: { userId: props.user?.id ?? null, theme }, intl: contextIntl },
    email,
    telegram,
  };

  useLayoutEffect(() => {
    form.current = mountCommentForm(root.current!, formProps, deps);
    return () => form.current?.destroy();
    // mounted once; the props and deps of every later render reach it through update below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const mounted = useRef(false);
  useLayoutEffect(() => {
    // the first render is the mount above; each one after it is a render of the parent
    if (mounted.current) {
      form.current?.update(formProps, deps);
    }
    mounted.current = true;
  });

  return <div ref={root} />;
}

/** the last textarea id given out, counted across every form on the page */
CommentForm.textareaCounter = 0;
