import { siteId, url } from 'common/settings';
import { LS_COLLAPSE_KEY } from 'common/constants';
import { setItem as localStorageSetItem, getItem as localStorageGetItem } from 'common/local-storage';
import { Comment } from 'common/types';

function getFromLocalStorage(): string[] {
  return JSON.parse(localStorageGetItem(LS_COLLAPSE_KEY) || '[]');
}

/**
 * returns list of serialized comments of type "site-id_url_comment-id"
 */
export const getCollapsedComments = (): string[] =>
  getFromLocalStorage().reduce((acc: string[], v: string) => {
    const components = v.split('_');
    if (components[0] === siteId && components[1] === url) {
      acc.push(components[2]);
    }
    return acc;
  }, []);

/**
 * @param info list of string of type "site-id_url_comment-id
 */
export const saveCollapsedComments = (siteId: string, url: string, info: Comment['id'][]): void => {
  const data = info.map((i) => `${siteId}_${url}_${i}`);
  const notForThisPost = getFromLocalStorage().filter((entry) => entry.indexOf(`${siteId}_${url}`) === -1);
  const all = new Set([...notForThisPost, ...data]);
  localStorageSetItem(LS_COLLAPSE_KEY, JSON.stringify([...all]));
};
