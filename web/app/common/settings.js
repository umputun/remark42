const querySettings = window.location.search.substr(1).split('&').reduce((acc, param) => {
  const pair = param.split('=');
  acc[pair[0]] = decodeURIComponent(pair[1]);
  return acc;
}, {}) || {};

export const siteId = querySettings['site_id'];
export const url = querySettings['url'];
export const maxShownComments = querySettings['max_shown_comments'];
