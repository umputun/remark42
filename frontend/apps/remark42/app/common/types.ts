export type User = {
  name: string;
  id: string;
  picture: string;
  ip: string;
  admin: boolean;
  block: boolean;
  verified: boolean;
  email_subscription?: boolean;
  paid_sub?: boolean;
};

/** data which is used on user-info page */
export type Profile = Pick<User, 'id' | 'name' | 'picture'>;

/**
 * What the widget posts to the page to open a profile. The page reads it, so it says nothing of the
 * reader: it names the author whose name was clicked, or nobody when readers open their own, and
 * carries the token the profile frame quotes when it asks the widget who the reader is
 */
export type ProfileRequest = Partial<Profile> & { token: string };

export interface BlockedUser {
  id: string;
  name: string;
  time: string;
}

export interface Locator {
  /** site id */
  site: string;
  /** post url */
  url: string;
}

export interface Comment {
  /** comment ID, read only */
  id: string;
  /** parent ID */
  pid: string;
  /** comment text, after md processing */
  text: string;
  /** original comment text */
  orig?: string;
  /** user info, read only */
  user: User;
  /** post locator */
  locator: Locator;
  /** comment score, read only */
  score: number;
  voted_ips: { Timestamp: string; Value: boolean }[];
  /**
   * vote delta,
   * if user hasn't voted delta will be 0,
   * -1/+1 for downvote/upvote
   */
  vote: 0 | 1 | -1;
  /** comment controversy, read only */
  controversy?: number;
  /** pointer to have empty default in json response */
  edit?: {
    time: string;
    summary: string;
  };
  /** time stamp, read only */
  time: string;
  /** pinned status, read only */
  pin?: boolean;
  /** delete status, read only */
  delete?: boolean;
  /** post title */
  title?: string;
  /**
   * @ClientOnly defines whether comments was hidden (deleted)
   *
   * Situation may occur for example if user decided to hide someone,
   * in this case we don't use `delete` field because comment with `delete`
   * still renders, and comment with `hidden` flag completely removed from DOM
   */
  hidden?: boolean;
}
export interface Node {
  comment: Comment;
  replies?: Node[];
}

export interface PostInfo {
  url: string;
  count: number;
  read_only?: boolean;
  first_time?: string;
  last_time?: string;
  /** comments after the returned page, set when the request asked for a limit */
  count_left?: number;
  /** the last top-level comment of the returned page, where the next page starts */
  last_comment?: string;
}

export interface Tree {
  comments: Node[];
  info: PostInfo;
  /** the pinned comments of the whole post, on the first page of a limited request only */
  pinned?: Comment[];
}

export type DefaultOAuthProvider =
  | 'apple'
  | 'facebook'
  | 'twitter'
  | 'google'
  | 'yandex'
  | 'github'
  | 'microsoft'
  | 'patreon'
  | 'discord'
  | 'telegram'
  | 'dev';
export type OAuthProvider = DefaultOAuthProvider | (string & {});
export type FormProvider = 'email' | 'anonymous';
export type Provider = OAuthProvider | FormProvider;

export interface Config {
  version: string;
  auth_providers: Provider[];
  edit_duration: number;
  admin_edit: boolean;
  max_comment_size: number;
  admins: string[];
  admin_email: string;
  low_score: number;
  critical_score: number;
  positive_score: boolean;
  readonly_age: number;
  max_image_size: number;
  simple_view: boolean;
  anon_vote: boolean;
  email_notifications: boolean;
  telegram_notifications: boolean;
  emoji_enabled: boolean;
}

export type Sorting = '-time' | '+time' | '-active' | '+active' | '-score' | '+score' | '-controversy' | '+controversy';
export type BlockTTL = 'permanently' | '43200m' | '10080m' | '1440m';
export type Theme = 'light' | 'dark';
/** a theme as the host page sets it: 'auto' follows the visitor's system preference */
export type ThemeSetting = Theme | 'auto';

/**
 * Comment component's edit mode:
 * whether it should have reply or edit Input shown
 */
export enum CommentMode {
  None,
  Reply,
  Edit,
}

/**
 * Used as api.uploadImage response type
 */
export interface Image {
  name: string;
  size: number;
  /** mime type of an image */
  type: string;
  url: string;
}

/** error struct returned in case of api call error */
export interface ApiError {
  /**
   * Error code, that is part of server error response.
   * Note that -1 is reserved for error where `error` field shall be used directly
   */
  code?: number;
  /** simple explanation */
  details?: string;
  /** in-depth explanation */
  error: string;
}

export interface EmailSubVerificationStatus {
  updated: boolean;
  address: string;
}
