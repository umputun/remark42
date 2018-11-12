const querySettings =
  window.location.search
    .substr(1)
    .split('&')
    .reduce((acc, param) => {
      const pair = param.split('=');
      acc[pair[0]] = decodeURIComponent(pair[1]);
      return acc;
    }, {}) || {};

export const siteId = remark_config.site_id;
export const url = remark_config.url;
export const maxShownComments = remark_config.max_shown_comments;
export const token = querySettings['token'];
