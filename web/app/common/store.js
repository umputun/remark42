let _instance;

class Store {
  constructor() {
    if (_instance) return _instance;

    this.data = {};

    _instance = this;

    return _instance;
  }

  set(key, obj) {
    this.data[key] = obj;
  }

  get(key) {
    return this.data[key];
  }

  addComment({ text, id, pid }) {
    const comments = this.data.comments;
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

    this.data.comments = this.data.comments.map(thread => paste(thread, newReply));
  }

  pasteComment(newComment) {
    this.data.comments = [newComment].concat(this.data.comments);
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

      thread.replies = thread.replies.map(reply => replace(reply, comment));

      return thread;
    };

    this.data.comments = this.data.comments.map(thread => replace(thread, newComment));
  }

  getPinnedComments() {
    return this.data.comments.reduce((acc, thread) => acc.concat(findPinnedComments(thread)), []);
  }
}

function findPinnedComments(thread) {
  if (thread.comment.pin) return [thread.comment];

  if (thread.replies) return thread.replies.reduce((acc, thread) => findPinnedComments(thread), []);

  return [];
}

export default new Store();
