import fetcher from './fetcher'

export const find = ({ url }) => fetcher.get(`/find?url=${url}&sort=time&format=tree`);

export const vote = ({ id, url, value }) => fetcher.get(`/vote/${id}?url=${url}&vote=${value}`);

export default {
  find,
  vote,
};
