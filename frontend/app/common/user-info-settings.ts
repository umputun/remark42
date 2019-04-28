import { UserInfo } from './types';

export const userInfo: Partial<UserInfo> =
  window.location.search
    .substr(1)
    .split('&')
    .reduce<{ [key: string]: string }>((acc, param) => {
      const pair = param.split('=');
      acc[pair[0]] = decodeURIComponent(pair[1]);
      return acc;
    }, {}) || {};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
if (((userInfo.isDefaultPicture as any) as string) !== '1') {
  userInfo.isDefaultPicture = false;
} else {
  userInfo.isDefaultPicture = true;
}

export const id = userInfo.id;
export const name = userInfo.name;
export const isDefaultPicture = userInfo.isDefaultPicture;
export const picture = userInfo.picture;
