import { getJsonItem, updateJsonItem } from 'common/local-storage';
import { LS_SAVED_COMMENT_VALUE } from 'common/constants';

export function getPersistedComments() {
  return getJsonItem<Record<string, string>>(LS_SAVED_COMMENT_VALUE);
}

export function getPersistedComment(id: string | undefined): string | undefined {
  const comments = getPersistedComments();
  if (!comments || !id) {
    return;
  }

  return comments[id];
}

export function updatePersistedComments(id: string, value: string) {
  updateJsonItem(LS_SAVED_COMMENT_VALUE, { ...getPersistedComments(), [id]: value });
}

export function removePersistedComment(id: string) {
  updateJsonItem<Record<string, string> | null>(LS_SAVED_COMMENT_VALUE, (data) => {
    if (!data) {
      return null;
    }
    delete data[id];
    return data;
  });
}
