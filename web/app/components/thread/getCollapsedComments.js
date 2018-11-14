import { LS_COLLAPSE_KEY } from 'common/constants';
import { getItem } from 'common/localStorage';

const getCollapsedComments = () => JSON.parse(getItem(LS_COLLAPSE_KEY) || '[]');

export default getCollapsedComments;
