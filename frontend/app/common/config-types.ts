import { UserInfo, Theme } from './types';

export interface CounterConfig {
  host: string;
  site_id: string;
  url?: string;
}

export type UserInfoConfig = UserInfo;

export interface CommentsConfig {
  host: string;
  site_id: string;
  url?: string;
  max_shown_comments?: number;
  theme?: Theme;
  page_title?: string;
  node?: string | HTMLElement;
  locale?: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  __colors__?: any;
}

export interface LastCommentsConfig {
  host: string;
  site_id: string;
  max_last_comments: number;
  locale?: string;
}
