import { h, Component } from 'preact';

import Comment from 'components/comment';

export default class Thread extends Component {
  constructor(props) {
    super(props);

    this.state = {
      hidden: false,
    };

    this.hide = this.hide.bind(this);
  }

  hide() {
    this.setState({ hidden: true });
  }

  render(props, { hidden }) {
    const { data: { comment, replies = [] }, mix, mods = {} } = props;

    return (
      <div className={b('thread', props, { hidden })}>
        <Comment
          data={comment}
          mods={{ level: mods.level }}
          onReply={props.onReply}
          onDelete={this.hide}
        />

        {
          !!replies.length && replies.map(thread => (
            <Thread
              data={thread}
              mods={{ level: mods.level < 5 ? mods.level + 1 : mods.level }}
              onReply={props.onReply}
            />
          ))
        }
      </div>
    );
  }
}
