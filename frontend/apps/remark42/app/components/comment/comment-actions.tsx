import { h } from 'preact';
import { useLayoutEffect, useRef } from 'preact/hooks';

import { useIntl } from 'common/intl';

import type { ActionsProps } from './actions.js';
import { mountActions } from './actions.js';

export type Props = ActionsProps;

/**
 * Mounts the plain comment actions into a div preact renders without props or children, so a
 * re-render of the comment leaves the div's attributes and content to the plain component.
 */
export function CommentActions(props: Props) {
  const intl = useIntl();
  const root = useRef<HTMLDivElement>(null);
  const actions = useRef<ReturnType<typeof mountActions> | null>(null);

  useLayoutEffect(() => {
    actions.current = mountActions(root.current!, props, intl);
    return () => actions.current?.destroy();
    // mounted once; the props and intl of every later render reach it through update below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // every render, so each button calls the handler the comment passed last
  useLayoutEffect(() => {
    actions.current?.update(props, intl);
  });

  return <div ref={root} />;
}
