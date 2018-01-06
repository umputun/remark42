import tmpl from 'blueimp-tmpl';

import { nodeId } from './settings';

let node = null;
let afterInit = null;

if (document.readyState !== 'complete') {
  window.addEventListener('DOMContentLoaded', initNode);
} else {
  initNode();
}

function initNode () {
  if (node) return;

  node = document.getElementById(nodeId);

  if (!node) return;

  if (afterInit) {
    afterInit();
  }
}

export default data => {
  // TODO: link to profile?
  // TODO: link to comment?
  // TODO: add photo?
  const templateComment = `
<div class="remark42__comment remark42__comment_level_{%= o.mods.level %} {%= o.mods.view ? ('remark42__comment_view_'  + o.mods.view) : '' %}">  
  <img src="{%= o.user.picture %}" alt="" class="remark42__avatar">
  
  <div class="remark42__content">
    <div class="remark42__info">
      <a href="#" class="remark42__username">{%= o.user.name %}</a>
      
      <span class="remark42__score">
        <a href="#" class="remark42__vote remark42__vote_type_up">vote up</a>
          <span class="remark42__score-sign">{%= o.scoreSign %}</span>{%= o.score %}
        <a href="#" class="remark42__vote remark42__vote_type_down">vote down</a>
      </span>
      
      <span class="remark42__time">{%= o.time %}</span>
    </div>
    
    <div class="remark42__text">
      {%# o.text %}
    </div>
  </div>    
</div>
  `;

  const renderComment = ({ comment, level }) => {
    const time = new Date(comment.time);
    // TODO: which format for datetime should we choose?
    // TODO: add smth that will count 'hours ago'
    // TODO: check out stash's impl
    const timeStr = `${time.toLocaleDateString()} ${time.toLocaleTimeString()}`;
    const data = {
      ...comment,
      scoreSign: comment.score > 0 ? '+' : (comment.score < 0 ? '−' : ''),
      score: Math.abs(comment.score),
      time: timeStr,
      mods: {
        level: level > 5 ? 5 : level,
        view: comment.user.admin ? 'admin' : '', // TODO: add default view or don't?
      },
    };

    return tmpl(templateComment, data);
  }

  const renderThread = ({ comment, replies, level }) => {
    let result = [renderComment({ comment, level })];

    if (replies) {
      result = result.concat(replies.map(thread => renderThread({ ...thread, level: level + 1 })));
    }

    return result.join('');
  };

  const render = () => {
    const result = data.comments.reduce((acc, thread) => acc.concat(renderThread({ ...thread, level: 0 })), []).join('');

    node.className = 'remark42';
    node.innerHTML = result;
  };

  if (node) {
    render();
  } else {
    afterInit = render;
  }
}
