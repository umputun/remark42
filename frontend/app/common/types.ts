export interface User {
  name: string;
  id: string;
  picture: string;
  ip: string;
  admin: boolean;
  block: boolean;
  verified: boolean;
  email_subscription?: boolean;
}

/** data which is used on user-info page */
export interface UserInfo {
  id: User['id'];
  name: string | '';
  isDefaultPicture: boolean;
  picture: string;
}

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
  vote: number;
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
   * Situatuon may occure for example if user decided to hide someone,
   * in this case we don't use `delete` field because comment with `delete`
   * still renders, and comment with `hidden` flag completely removed from DOM
   */
  hidden?: boolean;
}

export interface CommentsResponse {
  comments: Comment[];
  count: number;
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
}

export interface Tree {
  comments: Node[];
  info: PostInfo;
}

export type OAuthProvider = 'facebook' | 'twitter' | 'google' | 'yandex' | 'github' | 'microsoft' | 'dev';
export type FormProvider = 'email' | 'anonymous';
export type Provider = OAuthProvider | FormProvider;

export interface Config {
  version: string;
  auth_providers: Provider[];
  edit_duration: number;
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
  emoji_enabled: boolean;
}

export type Sorting = '-time' | '+time' | '-active' | '+active' | '-score' | '+score' | '-controversy' | '+controversy';

export type BlockTTL = 'permanently' | '43200m' | '10080m' | '1440m';

export interface BlockingDuration {
  label: string;
  value: BlockTTL;
}

export type Theme = 'light' | 'dark';

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
  code: number;
  /** simple explanation */
  details: string;
  /** in-depth explanation */
  error: string;
}
