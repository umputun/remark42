import { h } from 'preact';
import { useLayoutEffect, useRef } from 'preact/hooks';

import { useIntl } from 'common/intl';
import { useDispatch } from 'store/context';
import { patchComment } from 'store/comments/actions';
import { putCommentVote } from 'common/api';
import { StaticStore } from 'common/static-store';

import { mountVotes } from './votes.js';

type Props = {
  id: string;
  vote: 0 | -1 | 1;
  votes: number;
  controversy: number | undefined;
  disabled?: boolean;
};

/**
 * Mounts the plain vote buttons into a span preact renders without props or children, so a
 * re-render of the comment leaves the span's class and content to the plain component.
 */
export function CommentVotes(props: Props) {
  const intl = useIntl();
  const dispatch = useDispatch();
  const root = useRef<HTMLSpanElement>(null);
  const votes = useRef<ReturnType<typeof mountVotes> | null>(null);
  const deps: Parameters<typeof mountVotes>[2] = {
    intl,
    putVote: (id, vote) => putCommentVote({ id, vote }),
    patch: (patch) => dispatch(patchComment(patch)),
    positiveScore: () => Boolean(StaticStore.config.positive_score),
  };

  useLayoutEffect(() => {
    votes.current = mountVotes(root.current!, props, deps);
    return () => votes.current?.destroy();
    // mounted once; the props and deps of every later render reach it through update below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // every render, which costs a few property writes when nothing changed
  useLayoutEffect(() => {
    votes.current?.update(props, deps);
  });

  return <span ref={root} />;
}
