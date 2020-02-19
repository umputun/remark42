export const BASE_URL: string =
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  ((window as any).remark_config && (window as any).remark_config.host) || process.env.REMARK_URL!;
export const NODE_ID: string = process.env.REMARK_NODE!;
export const COMMENT_NODE_CLASSNAME_PREFIX = 'remark42__comment-';
