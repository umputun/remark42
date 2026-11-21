import type { FunctionComponent } from 'preact';
import { h } from 'preact';
import { shallowEqual } from 'store/context';
import { useCallback } from 'preact/hooks';
import clsx from 'clsx';

import { defineMessages, useIntl } from 'common/intl';
import type { Comment as CommentInterface } from 'common/types';
import { getHandleClickProps } from 'common/accessibility';
import type { StoreState } from 'store';
import { useAppDispatch, useAppSelector } from 'store';
import { setCollapse } from 'store/thread/actions';
import { getThreadIsCollapsed } from 'store/thread/getters';
import { InView } from 'components/root/in-view/in-view';
import { ConnectedComment as Comment } from 'components/comment/connected-comment';
import { CommentForm } from 'components/comment-form';

import styles from './thread.module.css';

interface Props {
  id: CommentInterface['id'];
  childs?: CommentInterface['id'][];
  level: number;

  getPreview(text: string): Promise<string>;
}

const commentSelector = (id: string) => (state: StoreState) => {
  const { theme, comments } = state;
  const { allComments, childComments } = comments;
  const comment = allComments[id];
  const childs = childComments[id];
  const collapsed = getThreadIsCollapsed(comment)(state);
  // a reply by a hidden author renders nothing, and a list must not be left without items
  const hasVisibleReplies = !!childs?.some((childId) => !allComments[childId]?.hidden);

  return { comment, childs, collapsed, theme, hasVisibleReplies };
};

export const Thread: FunctionComponent<Props> = ({ id, level, getPreview }) => {
  const dispatch = useAppDispatch();
  const intl = useIntl();
  const { collapsed, comment, childs, theme, hasVisibleReplies } = useAppSelector(commentSelector(id), shallowEqual);
  const collapse = useCallback(() => {
    dispatch(setCollapse(id, !collapsed));
  }, [id, collapsed, dispatch]);

  if (comment.hidden) return null;

  const indented = level > 0;
  const repliesCount = childs ? childs.length : 0;

  return (
    <div
      className={clsx(
        styles.root,
        indented && styles.indented,
        level === 6 && styles.level6,
        theme === 'dark' && styles.themeDark
      )}
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
            inView={inviewProps.inView}
          />
        )}
      </InView>

      {!collapsed && childs && hasVisibleReplies && (
        // the replies are items of a list of their own, which this item contains
        <div role="list">
          {childs.map((currentId) => (
            <Thread key={`thread-${currentId}`} id={currentId} level={Math.min(level + 1, 6)} getPreview={getPreview} />
          ))}
        </div>
      )}
      {level < 6 && (
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
