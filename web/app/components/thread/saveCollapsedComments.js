import { LS_COLLAPSE_KEY } from 'common/constants';

const saveCollapsedComments = comments => localStorage.setItem(LS_COLLAPSE_KEY, JSON.stringify(comments));

export default saveCollapsedComments;
