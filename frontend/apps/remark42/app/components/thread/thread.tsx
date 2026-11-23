import type { FunctionComponent } from 'preact';
import { h } from 'preact';
import { shallowEqual } from 'store/context';
import { useCallback, useContext } from 'preact/hooks';
import clsx from 'clsx';

import { defineMessages, useIntl } from 'common/intl';
import type { Comment as CommentInterface } from 'common/types';
import { getHandleClickProps } from 'common/accessibility';
import type { StoreState } from 'store';
import { useAppDispatch, useAppSelector } from 'store';
import { setCollapse } from 'store/thread/actions';
import { getThreadIsCollapsed } from 'store/thread/getters';
import { InView, RevealedComments } from 'components/root/in-view/in-view';
import { ConnectedComment as Comment } from 'components/comment/connected-comment';
import { CommentForm } from 'components/comment-form';

import styles from './thread.module.css';

interface Props {
  id: CommentInterface['id'];
  childs?: CommentInterface['id'][];
  level: number;

  getPreview(text: string): Promise<string>;
}

/** the deepest indent. Replies to a comment at this level are drawn at this level too */
const MAX_LEVEL = 6;

const noReplies: CommentInterface['id'][] = [];

/**
 * The comments a thread at `level` draws as items of its own list.
 *
 * Above the deepest indent these are the direct replies, and each draws its own replies inside
 * itself. The thread just above the deepest indent draws every reply below it, in thread order, as
 * one list, and a thread at the deepest indent draws none: all of them stand at one indent anyway,
 * and nested one inside another they would make the depth of the page's elements, and of the calls
 * that render them, follow the length of a chain of replies.
 */
function getReplies(state: StoreState, id: CommentInterface['id'], level: number): CommentInterface['id'][] {
  const { allComments, childComments } = state.comments;
  const childs = childComments[id];

  if (!childs || level >= MAX_LEVEL) return noReplies;
  if (level < MAX_LEVEL - 1) return childs;

  const replies: CommentInterface['id'][] = [];
  // the comments still to visit, the next one last
  const pending = childs.slice().reverse();

  while (pending.length > 0) {
    const replyId = pending.pop() as string;
    const reply = allComments[replyId];

    // a reply by a hidden author is not drawn, and neither is anything under it
    if (!reply || reply.hidden) continue;
    replies.push(replyId);

    const deeper = childComments[replyId];
    if (!deeper || getThreadIsCollapsed(reply)(state)) continue;
    for (let i = deeper.length - 1; i >= 0; i--) {
      pending.push(deeper[i]);
    }
  }

  return replies;
}

const threadSelector = (id: CommentInterface['id'], level: number) => (state: StoreState) => {
  const { theme, comments } = state;
  const { allComments, childComments } = comments;
  const comment = allComments[id];
  const repliesCount = childComments[id]?.length ?? 0;
  const collapsed = getThreadIsCollapsed(comment)(state);
  // a collapsed thread draws no replies, and neither does the thread of a hidden comment
  const replies = collapsed || comment.hidden ? noReplies : getReplies(state, id, level);
  // a reply by a hidden author renders nothing, and a list must not be left without items
  const hasVisibleReplies = replies.some((replyId) => !allComments[replyId]?.hidden);

  return { comment, repliesCount, collapsed, theme, replies, hasVisibleReplies };
};

type ThreadState = ReturnType<ReturnType<typeof threadSelector>>;

/**
 * `getReplies` builds a list anew on every change of the store, so the lists are compared by their
 * content, and everything else as `shallowEqual` does
 */
function sameThread({ replies: a, ...restA }: ThreadState, { replies: b, ...restB }: ThreadState): boolean {
  return (
    shallowEqual(restA, restB) && (a === b || (a.length === b.length && a.every((replyId, i) => replyId === b[i])))
  );
}

export const Thread: FunctionComponent<Props> = ({ id, level, getPreview }) => {
  const dispatch = useAppDispatch();
  const intl = useIntl();
  const { collapsed, comment, repliesCount, theme, replies, hasVisibleReplies } = useAppSelector(
    threadSelector(id, level),
    sameThread
  );
  const revealed = useContext(RevealedComments).has(id);
  const collapse = useCallback(() => {
    dispatch(setCollapse(id, !collapsed));
  }, [id, collapsed, dispatch]);

  if (comment.hidden) return null;

  const indented = level > 0;

  return (
    <div
      className={clsx(styles.root, indented && styles.indented, theme === 'dark' && styles.themeDark)}
      role="listitem"
    >
      <InView>
        {(inviewProps) => (
          <Comment
            CommentForm={CommentForm}
            ref={inviewProps.ref}
            key={`comment-${id}`}
            view="main"
            intl={intl}
            data={comment}
            repliesCount={repliesCount}
            level={level}
            inView={inviewProps.inView || revealed}
          />
        )}
      </InView>

      {!collapsed && hasVisibleReplies && (
        // the replies are items of a list of their own, which this item contains
        <div role="list">
          {replies.map((currentId) => (
            <Thread key={`thread-${currentId}`} id={currentId} level={level + 1} getPreview={getPreview} />
          ))}
        </div>
      )}
      {level < MAX_LEVEL && (
        <div
          className={collapsed ? styles.collapsed : styles.collapse}
          {...getHandleClickProps(collapse)}
          aria-label={intl.formatMessage(messages.toggleThread)}
          aria-expanded={!collapsed}
        >
          <div />
        </div>
      )}
    </div>
  );
};

const messages = defineMessages({
  toggleThread: {
    id: 'thread.toggle',
    defaultMessage: 'Collapse or expand thread',
  },
});
