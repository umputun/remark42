// we know that `comments` is defined because this module was loaded by `load-module`
type CommentsModuleConfig = Omit<RawConfig, 'comments'> & { comments: Record<keyof CommentsModuleParams, unknown> };
const { params, comments: moduleParams } = window.remark42 as CommentsModuleConfig;

const iframe = document.createElement('iframe');

if (typeof moduleParams.container !== 'string') {
  throw new Error(`wasn't able to find container in DOM`);
}

const container = document.querySelector(moduleParams.container);
