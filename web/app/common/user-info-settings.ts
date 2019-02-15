import { UserInfo } from './types';

export const userInfo: Partial<UserInfo> =
  window.location.search
    .substr(1)
    .split('&')
    .reduce((acc, param) => {
      const pair = param.split('=');
      (acc as any)[pair[0]] = decodeURIComponent(pair[1]);
      return acc;
    }, {}) || {};

if (((userInfo.isDefaultPicture as any) as string) !== '1') {
  userInfo.isDefaultPicture = false;
} else {
  userInfo.isDefaultPicture = true;
}

export const id = userInfo.id;
export const name = userInfo.name;
export const isDefaultPicture = userInfo.isDefaultPicture;
export const picture = userInfo.picture;
