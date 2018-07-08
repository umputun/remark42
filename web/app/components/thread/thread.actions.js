export const THREAD_SET_COLLAPSE = 'THREAD/COLLAPSE_SET';
export const setCollapse = (comment, collapsed) => ({
  type: THREAD_SET_COLLAPSE,
  comment,
  collapsed,
});
