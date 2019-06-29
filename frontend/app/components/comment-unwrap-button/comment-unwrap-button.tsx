/** @jsx h */

import { h, FunctionalComponent } from 'preact';
import { Comment } from '@app/common/types';
import { BoundActionCreator } from '@app/utils/actionBinder';
import { unwrapNewComments } from '@app/store/comments/actions';
import { getHandleClickProps } from '@app/common/accessibility';

function pluralize(count: number): string {
  if (count % 10 === 1) {
    if (count % 100 === 11) return `${count} new replies`;
    return `${count} new reply`;
  }
  return `${count} new replies`;
}

export const CommentUnwrapButton: FunctionalComponent<{
  className?: string;
  id: Comment['id'];
  count: number;
  unwrapComment: BoundActionCreator<typeof unwrapNewComments>;
}> = props => (
  <span className={props.className} {...getHandleClickProps(() => props.unwrapComment(props.id))}>
    ... +{pluralize(props.count)}
  </span>
);
