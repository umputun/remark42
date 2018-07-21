export const saveTimeDiff = (comments, timeDiff) => {
  comments.forEach(comment => {
    comment.comment.timeDiff = timeDiff;
    if (comment.replies) {
      saveTimeDiff(comment.replies, timeDiff);
    }
  });
};
