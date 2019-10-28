/** @jsx createElement */
import { createElement, RenderableProps, FunctionComponent } from 'preact';
import { useStore } from 'react-redux';
import b from 'bem-react-helper';

import { ConnectedComment as Comment } from '@app/components/comment/connected-comment';
import { Comment as CommentInterface } from '@app/common/types';
import { getThreadIsCollapsed } from '@app/store/thread/getters';
import { StoreState } from '@app/store';
import { InView } from '../root/in-view/in-view';

const mapStateToProps = (state: StoreState, props: { id: CommentInterface['id'] }) => {
  const comment = state.comments[props.id];
  return {
    comment,
    childs: state.childComments[props.id],
    collapsed: getThreadIsCollapsed(comment)(state),
    isCommentsDisabled: !!state.info.read_only,
    theme: state.theme,
  };
};

interface OwnProps {
  id: CommentInterface['id'];
  childs?: (CommentInterface['id'])[];
  level: number;
  mix?: string;

  getPreview(text: string): Promise<string>;
}

type Props = OwnProps & ReturnType<typeof mapStateToProps>;

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
      <InView>
        {inviewProps => (
          <Comment
            ref={ref => inviewProps.ref(ref)}
            key={`comment-${props.id}`}
            view="main"
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
        childs.map(id => (
          <ConnectedThread key={`thread-${id}`} id={id} level={Math.min(level + 1, 6)} getPreview={props.getPreview} />
        ))}
    </div>
  );
}

export const ConnectedThread: FunctionComponent<OwnProps> = props => {
  const providedProps = mapStateToProps(useStore().getState(), props);
  return <Thread {...props} {...providedProps} />;
};
