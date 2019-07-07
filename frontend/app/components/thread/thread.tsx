/** @jsx h */
import { h, RenderableProps } from 'preact';
import { connect } from 'preact-redux';
import b from 'bem-react-helper';

import { ConnectedComment as Comment } from '@app/components/comment/connected-comment';
import { Comment as CommentInterface } from '@app/common/types';
import { getThreadIsCollapsed } from '@app/store/thread/getters';
import { StoreState } from '@app/store';

const mapStateToProps = (state: StoreState, props: { id: CommentInterface['id'] }) => {
  const comment = state.comments[props.id];
  return {
    comment,
    childs: state.childComments[props.id],
    collapsed: getThreadIsCollapsed(state, comment),
    isCommentsDisabled: !!state.info.read_only,
    theme: state.theme,
  };
};

type Props = {
  id: CommentInterface['id'];
  childs?: (CommentInterface['id'])[];
  level: number;
  mix?: string;

  getPreview(text: string): Promise<string>;
} & ReturnType<typeof mapStateToProps>;

function Thread(props: RenderableProps<Props>) {
  const { collapsed, comment, childs, level, theme } = props;

  if (comment.hidden) return null;

  const indented = level > 0;
  const repliesCount = childs ? childs.length : 0;

  return (
    <div
      className={b('thread', props, { level, theme, indented })}
      role={['listitem'].concat(!collapsed && !!repliesCount ? 'list' : []).join(' ')}
      aria-expanded={!collapsed}
    >
      <Comment key={`comment-${props.id}`} view="main" data={comment} repliesCount={repliesCount} level={level} />

      {!collapsed &&
        childs &&
        !!childs.length &&
        childs.map(id => (
          <ConnectedThread key={`thread-${id}`} id={id} level={Math.min(level + 1, 6)} getPreview={props.getPreview} />
        ))}
    </div>
  );
}

export const ConnectedThread = connect(mapStateToProps)(Thread);
