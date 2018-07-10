/** @jsx h */
import { h, Component } from 'preact';
import { connect } from 'preact-redux';

import Comment from 'components/comment';
import { setCollapse } from './thread.actions';
import { getThreadIsCollapsed } from './thread.getters';

class Thread extends Component {
  constructor(props) {
    super(props);
    this.onCollapseToggle = this.onCollapseToggle.bind(this);
  }

  onCollapseToggle() {
    this.props.setCollapse(this.props.data.comment, !this.props.collapsed);
  }

  render(props) {
    const {
      collapsed,
      data: { comment, replies = [] },
      mods = {},
    } = props;

    return (
      <div
        className={b('thread', props)}
        role={['listitem'].concat(!collapsed && replies.length ? 'list' : [])}
        aria-expanded={!collapsed}
      >
        <Comment
          data={comment}
          mods={{ level: mods.level, collapsed }}
          onReply={props.onReply}
          onEdit={props.onEdit}
          onCollapseToggle={this.onCollapseToggle}
        />

        {!collapsed &&
          !!replies.length &&
          replies.map(thread => (
            <ConnectedThread
              key={thread.comment.id}
              data={thread}
              mods={{ level: mods.level < 5 ? mods.level + 1 : mods.level }}
              onReply={props.onReply}
              onEdit={props.onEdit}
            />
          ))}
      </div>
    );
  }
}

const ConnectedThread = connect(
  (state, props) => ({ collapsed: getThreadIsCollapsed(state, props.data.comment) }),
  { setCollapse }
)(Thread);

export default ConnectedThread;
