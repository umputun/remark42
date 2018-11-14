import { LS_COLLAPSE_KEY } from 'common/constants';
import { setItem } from 'common/localStorage';

const saveCollapsedComments = comments => setItem(LS_COLLAPSE_KEY, JSON.stringify(comments));

export default saveCollapsedComments;
