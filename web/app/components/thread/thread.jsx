import { h, Component } from 'preact';

import { LS_COLLAPSE_KEY } from 'common/constants';
import { siteId, url } from 'common/settings';

import Comment from 'components/comment';

export default class Thread extends Component {
  constructor(props) {
    super(props);

    if (this.props.data && this.props.data.comment) {
      this.updateCollapsedState(this.props.data.comment);
    }

    this.onCollapseToggle = this.onCollapseToggle.bind(this);
  }

  componentWillReceiveProps(nextProps) {
    if (nextProps.data && nextProps.data.comment) {
      this.updateCollapsedState(nextProps.data.comment);
    }
  }

  updateCollapsedState(comment) {
    this.lsCollapsedID = `${siteId}_${url}_${comment.id}`;

    this.state = {
      collapsed: getCollapsedComments().includes(this.lsCollapsedID),
    };
  }

  onCollapseToggle() {
    const collapsed = !this.state.collapsed;

    this.setState({ collapsed: !this.state.collapsed });

    let collapsedComments = getCollapsedComments();

    if (collapsed) {
      if (!collapsedComments.includes(this.lsCollapsedID)) {
        collapsedComments = collapsedComments.concat(this.lsCollapsedID);
      }
    } else {
      collapsedComments = collapsedComments.filter(id => id !== this.lsCollapsedID);
    }

    saveCollapsedComments(collapsedComments);
  }

  render(props, { collapsed }) {
    const { data: { comment, replies = [] }, mix, mods = {} } = props;

    return (
      <div
        className={b('thread', props)}
        role={['listitem'].concat(!collapsed && replies.length ? 'list' : [])}
        aria-expanded={!collapsed}
      >
        <Comment
          data={comment}
          mods={{ level: mods.level, collapsed, collapsible: !!replies.length }}
          onReply={props.onReply}
          onEdit={props.onEdit}
          onCollapseToggle={this.onCollapseToggle}
        />

        {
          !collapsed && !!replies.length && replies.map(thread => (
            <Thread
              data={thread}
              mods={{ level: mods.level < 5 ? mods.level + 1 : mods.level }}
              onReply={props.onReply}
              onEdit={props.onEdit}
            />
          ))
        }
      </div>
    );
  }
}

function getCollapsedComments() {
  return JSON.parse(localStorage.getItem(LS_COLLAPSE_KEY) || '[]');
}

function saveCollapsedComments(comments) {
  localStorage.setItem(LS_COLLAPSE_KEY, JSON.stringify(comments));
}
