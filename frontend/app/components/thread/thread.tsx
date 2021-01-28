import { h, FunctionComponent } from 'preact';
import { useSelector, useDispatch, shallowEqual } from 'react-redux';
import { useCallback } from 'preact/hooks';
import b from 'bem-react-helper';
import { useIntl } from 'react-intl';

import { Comment as CommentInterface } from 'common/types';
import { getHandleClickProps } from 'common/accessibility';
import { StoreState } from 'store';
import { setCollapse } from 'store/thread/actions';
import { getThreadIsCollapsed } from 'store/thread/getters';
import InView from 'components/root/in-view/in-view';
import { ConnectedComment as Comment } from 'components/comment/connected-comment';
import { CommentForm } from 'components/comment-form';

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
  }, [id, collapsed, dispatch]);

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

      {!collapsed &&
        childs &&
        !!childs.length &&
        childs.map((currentId) => (
          <Thread key={`thread-${currentId}`} id={currentId} level={Math.min(level + 1, 6)} getPreview={getPreview} />
        ))}
      {level < 6 && (
        <div className={b('thread__collapse', { mods: { collapsed } })} {...getHandleClickProps(collapse)}>
          <div />
        </div>
      )}
    </div>
  );
};
