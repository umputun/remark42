import { h } from 'preact';
import { useLayoutEffect, useRef } from 'preact/hooks';

import type { IntlShape } from 'common/intl';
import type { User, Theme, PostInfo } from 'common/types';
import type { StoreState } from 'store';
import { useSignInDeps } from 'components/auth/auth';

import { mountAuthPanel } from './panel.js';

interface OwnProps {
  user: User | null;
  hiddenUsers: StoreState['hiddenUsers'];
  isCommentsDisabled: boolean;
  postInfo: PostInfo;

  signout(): Promise<void>;
  onCommentsChangeReadOnlyMode(readOnly: boolean): Promise<void>;
  onBlockedUsersShow(): void;
  onBlockedUsersHide(): void;
}

export interface Props extends OwnProps {
  intl: IntlShape;
  theme: Theme;
}

/**
 * Mounts the plain auth panel into a div preact renders without props or children, so a
 * re-render of the page leaves the div's attributes and content to the plain component.
 */
export function AuthPanel(props: OwnProps) {
  const deps = useSignInDeps();
  const root = useRef<HTMLDivElement>(null);
  const panel = useRef<ReturnType<typeof mountAuthPanel> | null>(null);

  useLayoutEffect(() => {
    panel.current = mountAuthPanel(root.current!, props, deps);
    return () => panel.current?.destroy();
    // mounted once; the props and deps of every later render reach it through update below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // every render, so each button calls the handler passed last
  useLayoutEffect(() => {
    panel.current?.update(props, deps);
  });

  return <div ref={root} />;
}
