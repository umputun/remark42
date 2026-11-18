import { BASE_URL } from './constants';

/**
 * Address of a file the server embeds from `backend/app/webassets/assets` and serves under `/web`.
 * The bundle does not build these files, so their url is absolute to the instance.
 */
export function webAssetUrl(name: string): string {
  return `${BASE_URL}/web/${name}`;
}
