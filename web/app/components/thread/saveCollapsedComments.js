import { LS_COLLAPSE_KEY } from 'common/constants';
import { setItem as localStorageSetItem } from 'common/localStorage';

const saveCollapsedComments = comments => localStorageSetItem(LS_COLLAPSE_KEY, JSON.stringify(comments));

export default saveCollapsedComments;
