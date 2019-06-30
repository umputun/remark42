/** @jsx h */
import { h, RenderableProps } from 'preact';
import { connect } from 'preact-redux';
import b from 'bem-react-helper';

import { ConnectedComment as Comment } from '@app/components/comment/connected-comment';
import { Node, Theme } from '@app/common/types';
import { getThreadIsCollapsed } from '@app/store/thread/getters';
import { StoreState } from '@app/store';

interface Props {
  collapsed: boolean;
  data: Node;
  isCommentsDisabled: boolean;
  level: number;
  theme: Theme;
  mix?: string;

  getPreview(text: string): Promise<string>;
}

function Thread(props: RenderableProps<Props>) {
  const {
    collapsed,
    data: { comment, replies = [] },
    level,
    theme,
  } = props;

  const indented = level > 0;

  return (
    <div
      className={b('thread', props, { level, theme, indented })}
      role={['listitem'].concat(!collapsed && replies.length ? 'list' : []).join(' ')}
      aria-expanded={!collapsed}
    >
      <Comment view="main" data={comment} repliesCount={replies.length} level={level} />

      {!collapsed &&
        !!replies.length &&
        replies.map(thread => (
          <ConnectedThread
            key={thread.comment.id}
            data={thread}
            level={Math.min(level + 1, 6)}
            getPreview={props.getPreview}
          />
        ))}
    </div>
  );
}

export const ConnectedThread = connect((state: StoreState, props: { data: Node }) => ({
  collapsed: getThreadIsCollapsed(state, props.data.comment),
  isCommentsDisabled: !!state.info.read_only,
  theme: state.theme,
}))(Thread);
