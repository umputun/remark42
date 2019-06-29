/** @jsx h */
import { h, RenderableProps } from 'preact';
import { connect } from 'preact-redux';
import b from 'bem-react-helper';

import { ConnectedComment as Comment } from '@app/components/comment/connected-comment';
import { Node, Theme } from '@app/common/types';
import { getThreadIsCollapsed } from '@app/store/thread/getters';
import { StoreState } from '@app/store';
import { CommentUnwrapButton } from '../comment-unwrap-button/comment-unwrap-button';
import { bindActions } from '@app/utils/actionBinder';
import { unwrapNewComments } from '@app/store/comments/actions';

const boundActions = bindActions({
  unwrapNewComments,
});

type Props = {
  collapsed: boolean;
  data: Node;
  isCommentsDisabled: boolean;
  level: number;
  theme: Theme;
  mix?: string;

  getPreview(text: string): Promise<string>;
} & typeof boundActions;

function Thread(props: RenderableProps<Props>) {
  const {
    collapsed,
    data: { comment, replies = [] },
    level,
    theme,
    unwrapNewComments,
  } = props;

  const indented = level > 0;

  let newRepliesCount = 0;
  const filteredReplies = replies.filter(thread => {
    if (thread.comment.new) {
      ++newRepliesCount;
      return false;
    }
    return true;
  });

  return (
    <div
      className={b('thread', props, { level, theme, indented })}
      role={['listitem'].concat(!collapsed && replies.length ? 'list' : []).join(' ')}
      aria-expanded={!collapsed}
    >
      <Comment view="main" data={comment} repliesCount={replies.length} level={level} />

      {!collapsed &&
        !!filteredReplies.length &&
        filteredReplies.map(thread => (
          <ConnectedThread
            key={thread.comment.id}
            data={thread}
            level={Math.min(level + 1, 6)}
            getPreview={props.getPreview}
          />
        ))}

      {!!newRepliesCount && (
        <CommentUnwrapButton
          className="thread__action thread__unwrap-new-replies-action"
          count={newRepliesCount}
          id={comment.id}
          unwrapComment={unwrapNewComments}
        />
      )}
    </div>
  );
}

const mapStateToProps = (state: StoreState, props: { data: Node }) => ({
  collapsed: getThreadIsCollapsed(state, props.data.comment),
  theme: state.theme,
  isCommentsDisabled: !!state.info.read_only,
});

export const ConnectedThread = connect(
  mapStateToProps,
  boundActions
)(Thread);
