import { LS_COLLAPSE_KEY } from 'common/constants';

const getCollapsedComments = () => JSON.parse(localStorage.getItem(LS_COLLAPSE_KEY) || '[]');

export default getCollapsedComments;
