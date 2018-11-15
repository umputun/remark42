import { LS_COLLAPSE_KEY } from 'common/constants';
import { getItem as localStorageGetItem } from 'common/localStorage';

const getCollapsedComments = () => JSON.parse(localStorageGetItem(LS_COLLAPSE_KEY) || '[]');

export default getCollapsedComments;
