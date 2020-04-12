/** @jsx createElement */
import { createElement, FunctionComponent } from 'preact';
import { useSelector, useDispatch, shallowEqual } from 'react-redux';
import { useCallback } from 'preact/hooks';
import b from 'bem-react-helper';

import { Comment as CommentInterface } from '@app/common/types';
import { getHandleClickProps } from '@app/common/accessibility';
import { StoreState } from '@app/store';
import { setCollapse } from '@app/store/thread/actions';
import { getThreadIsCollapsed } from '@app/store/thread/getters';
import { InView } from '@app/components/root/in-view/in-view';
import { ConnectedComment as Comment } from '@app/components/comment/connected-comment';
import { CommentForm } from '@app/components/comment-form';
import { useIntl } from 'react-intl';

interface Props {
  id: CommentInterface['id'];
  childs?: CommentInterface['id'][];
  level: number;
  mix?: string;

  getPreview(text: string): Promise<string>;
}

const commentSelector = (id: string) => (state: StoreState) => {
  const { theme, comments } = state;
  const { allComments, childComments } = comments;
  const comment = allComments[id];
  const childs = childComments[id];
  const collapsed = getThreadIsCollapsed(comment)(state);

  return { comment, childs, collapsed, theme };
};

export const Thread: FunctionComponent<Props> = ({ id, level, mix, getPreview }) => {
  const dispatch = useDispatch();
  const intl = useIntl();
  const { collapsed, comment, childs, theme } = useSelector(commentSelector(id), shallowEqual);
  const collapse = useCallback(() => {
    dispatch(setCollapse(id, !collapsed));
  }, [id, collapsed]);

  if (comment.hidden) return null;

  const indented = level > 0;
  const repliesCount = childs ? childs.length : 0;

  return (
    <div
      className={b('thread', { mix }, { level, theme, indented })}
      role={['listitem'].concat(!collapsed && !!repliesCount ? 'list' : []).join(' ')}
      aria-expanded={!collapsed}
    >
      <InView>
        {inviewProps => (
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

      {!collapsed &&
        childs &&
        !!childs.length &&
        childs.map(currentId => (
          <Thread key={`thread-${currentId}`} id={currentId} level={Math.min(level + 1, 6)} getPreview={getPreview} />
        ))}
      {level < 6 && (
        <div
          className={b('thread__collapse', { mods: { collapsed } })}
          role="button"
          {...getHandleClickProps(collapse)}
        >
          <div></div>
        </div>
      )}
    </div>
  );
};
