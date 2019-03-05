import { Comment, Node, User } from '@app/common/types';

/**
 * Traverses through tree and applies function to comment with given id.
 * Note that function must not mutate comment, or rerender will not happen
 */
function mapTreeIfID(tree: Node[], id: Comment['id'], fn: (c: Node) => Node): Node[] {
  // path of indexes to comment with given id
  let path: number[] = [];
  const subfn = (tree: Node[], level: number): boolean => {
    for (let i = 0; i < tree.length; i++) {
      path = path.slice(0, level);
      path.push(i);
      if (id === tree[i].comment.id) return true;
      if (tree[i].replies) {
        if (subfn(tree[i].replies!, level + 1)) return true;
      }
    }
    return false;
  };
  if (!subfn(tree, 0)) return tree;

  // dereferencing (cloning) node path to comment with id,
  // so react will cause rerender
  const treeClone = [...tree];
  let subtree = treeClone;
  for (let i = 0; i < path.length; i++) {
    const index = path[i];
    if (i === path.length - 1) {
      subtree[index] = fn(subtree[index]);
      break;
    }
    subtree[index] = { comment: subtree[index].comment, replies: [...subtree[index].replies!] };
    subtree = subtree[index].replies!;
  }

  return treeClone;
}

/**
 * Traverses through tree and applies function to comment on which function passed.
 * Note that function must not mutate comment
 */
function mapTree(tree: Node[], fn: (c: Comment) => Comment): Node[] {
  return tree.map(node => {
    const clone: Node = {
      comment: fn(node.comment),
    };
    if (node.replies) {
      clone.replies = mapTree(node.replies, fn);
    }
    return clone;
  });
}

export function findPinnedComments(thread: Node): Comment[] {
  let result: Comment[] = [];

  if (thread.comment.pin) {
    result = result.concat(thread.comment);
  }

  if (thread.replies) {
    result = result.concat(
      thread.replies.reduce((acc: Comment[], thread: Node) => acc.concat(findPinnedComments(thread)), [])
    );
  }

  return result;
}

export function getPinnedComments(threads: Node[]): Comment[] {
  return threads.reduce((acc: Comment[], thread: Node) => acc.concat(findPinnedComments(thread)), []);
}

export function removeComment(comments: Node[], id: Comment['id']): Node[] {
  return mapTreeIfID(
    comments,
    id,
    (n): Node => ({
      comment: {
        ...n.comment,
        delete: true,
      },
      replies: n.replies,
    })
  );
}

export function setCommentPin(comments: Node[], id: Comment['id'], value: boolean): Node[] {
  return mapTreeIfID(
    comments,
    id,
    (n): Node => ({
      comment: {
        ...n.comment,
        pin: value,
      },
      replies: n.replies,
    })
  );
}

export function setUserVerified(comments: Node[], userId: User['id'], value: boolean): Node[] {
  return mapTree(comments, comment => {
    if (comment.user.id !== userId) return comment;
    return {
      ...comment,
      user: {
        ...comment.user,
        verified: value,
      },
    };
  });
}

function pasteReply(comments: Node[], reply: Comment): Node[] {
  return mapTreeIfID(
    comments,
    reply.pid,
    (n): Node => {
      const nn = { ...n };
      if (!nn.replies) nn.replies = [];
      nn.replies = [{ comment: reply }, ...nn.replies];
      return nn;
    }
  );
}

export function addComment(comments: Node[], comment: Comment): Node[] {
  if (comment.pid !== '') {
    return pasteReply(comments, comment);
  }
  return [{ comment }, ...comments];
}

export function replaceComment(comments: Node[], comment: Comment): Node[] {
  return mapTreeIfID(comments, comment.id, n => ({ ...n, comment }));
}

export function delay(ms: number = 100): Promise<void> {
  return new Promise(resolve => {
    setTimeout(resolve, ms);
  });
}
