import { Component } from 'preact';

export class CommentInfo extends Component {
  constructor(props) {
    super(props);
  }

  render(props) {
    return (
      <div className="comment__info">
        {
          mods.view !== 'user' && (
            <img
              className={b('comment__avatar', {}, { default: o.user.isDefaultPicture })}
              src={o.user.isDefaultPicture ? require('./__avatar/comment__avatar.svg') : o.user.picture}
              alt=""
            />
          )
        }

        {
          mods.view !== 'user' && (
            <span
              className="comment__username"
              title={o.user.id}
              onClick={this.toggleUserInfoVisibility}
            >{o.user.name}</span>
          )
        }

        {
          isAdmin && mods.view !== 'user' && (
            <span
              onClick={() => this.toggleVerify(o.user.verified)}
              aria-label="Toggle verification"
              title={o.user.verified ? 'Verified user' : 'Unverified user'}
              className={b('comment__verification', {}, { active: o.user.verified, clickable: true })}
            />
          )
        }

        {
          !isAdmin && !!o.user.verified && mods.view !== 'user' && (
            <span
              title="Verified user"
              className={b('comment__verification', {}, { active: true })}
            />
          )
        }

        <a href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`} className="comment__time">{o.time}</a>

        {
          mods.level > 0 && mods.view !== 'user' && (
            <a
              className="comment__link-to-parent"
              href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.pid}`}
              aria-label="Go to parent comment"
              title="Go to parent comment"
              onClick={this.scrollToParent}
            />
          )
        }

        {
          isAdmin && userBlocked && mods.view !== 'user' && (
            <span className="comment__status">Blocked</span>
          )
        }

        {
          isAdmin && !userBlocked && deleted && (
            <span className="comment__status">Deleted</span>
          )
        }

        {
          !mods.disabled && mods.view !== 'user' && (
            <span
              className={b('comment__action', {}, { type: 'collapse', selected: mods.collapsed })}
              tabIndex="0"
              onClick={this.toggleCollapse}
            >{mods.collapsed ? '+' : '−'}</span>
          )
        }

        <span className={b('comment__score', {}, { view: o.score.view })}>
          <span
            className={b('comment__vote', {}, { type: 'up', selected: scoreIncreased, disabled: isGuest || isCurrentUser })}
            role="button"
            aria-disabled={isGuest || isCurrentUser}
            tabIndex="0"
            onClick={isGuest || isCurrentUser ? null : this.increaseScore}
            title={isGuest ? 'Only authorized users are allowed to vote' : (isCurrentUser ? 'You can\'t vote for your own comment' : null)}
          >Vote up</span>

          <span className="comment__score-value">
            {o.score.sign}{o.score.value}
          </span>


          <span
            className={b('comment__vote', {}, { type: 'down', selected: scoreDecreased, disabled: isGuest || isCurrentUser })}
            role="button"
            aria-disabled={isGuest || isCurrentUser ? 'true' : 'false'}
            tabIndex="0"
            onClick={isGuest || isCurrentUser ? null : this.decreaseScore}
            title={isGuest ? 'Only authorized users are allowed to vote' : (isCurrentUser ? 'You can\'t vote for your own comment' : null)}
          >Vote down</span>
        </span>
      </div>);
  }
}