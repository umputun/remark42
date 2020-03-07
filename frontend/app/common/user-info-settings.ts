import { UserInfo } from './types';
import parseQuery from '@app/utils/parseQuery';

export const userInfo: Partial<UserInfo> = parseQuery();

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const isDefaultPicture = ((userInfo.isDefaultPicture as any) as string) !== '1';
export const id = userInfo.id;
export const name = userInfo.name;
export const picture = userInfo.picture;
