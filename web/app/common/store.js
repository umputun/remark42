let _instance;

class Store {
  constructor() {
    if (_instance) return _instance;

    this.data = {};

    this.listeners = {};

    _instance = this;

    return _instance;
  }

  onUpdate(key, cb) {
    if (!this.listeners[key]) this.listeners[key] = [];

    this.listeners[key].push(cb);
  }


  set(key, obj) {
    this.data[key] = obj;

    if (this.listeners[key]) this.listeners[key].forEach(cb => cb(this.data[key]));
  }

  get(key) {
    return this.data[key];
  }

  addComment({ text, id, pid }) {
    const newComment = {
      comment: {
        id,
        text,
        user: this.get('user'),
        time: new Date(),
        ...(pid ? { pid } : {}),
      },
    };

    if (pid) {
      this.pasteReply(newComment);
    } else {
      this.pasteComment(newComment);
    }
  }

  pasteReply(newReply) {
    let again = true;

    const concatReply = (root, reply) => {
      root.replies = root.replies || [];
      root.replies = [reply].concat(root.replies);

      again = false;

      return root;
    }

    const paste = (root, commentObj) => {
      if (!again) return root;

      if (root.comment.id === commentObj.comment.pid) {
        return concatReply(root, commentObj);
      }

      if (root.replies) {
        root.replies = root.replies.map(reply => {
          if (reply.comment.id === commentObj.comment.pid) {
            return concatReply(reply, commentObj);
          } else {
            return paste(reply, commentObj);
          }
        })
      }

      return root;
    };

    this.set('comments', this.data.comments.map(thread => paste(thread, newReply)));
  }

  pasteComment(newComment) {
    this.set('comments', [newComment].concat(this.data.comments));
  }

  replaceComment(newComment) {
    let again = true;

    const replace = (thread, comment) => {
      if (!again) return thread;

      if (thread.comment.id === comment.id) {
        thread.comment = comment;
        again = false;
        return thread;
      }

      if (thread.replies) {
        thread.replies = thread.replies.map(reply => replace(reply, comment));
      }

      return thread;
    };

    this.set('comments', this.data.comments.map(thread => replace(thread, newComment)));
  }

  getPinnedComments() {
    return this.data.comments.reduce((acc, thread) => acc.concat(findPinnedComments(thread)), []);
  }
}

function findPinnedComments(thread) {
  let result = [];

  if (thread.comment.pin) {
    result = result.concat(thread.comment);
  }

  if (thread.replies) {
    result = result.concat(thread.replies.reduce((acc, thread) => acc.concat(findPinnedComments(thread)), []));
  }

  return result;
}

export default new Store();
