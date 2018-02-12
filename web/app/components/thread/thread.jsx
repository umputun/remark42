import { h, Component } from 'preact';

import Comment from 'components/comment';

export default class Thread extends Component {
  render(props) {
    const { data: { comment, replies = [] }, mix, mods = {} } = props;

    return (
      <div className={b('thread', props)}>
        <Comment
          data={comment}
          mods={{ level: mods.level }}
          onReply={props.onReply}
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
