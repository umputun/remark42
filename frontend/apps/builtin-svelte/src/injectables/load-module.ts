const config = window.remark42;
const components = [];

if (!config) {
  throw new Error('remark42 config is not defined');
}

const { params } = config;

if (!params) {
  throw new Error('remark42 params is not defined');
}

if (typeof params.host !== 'string') {
  throw new Error('host is not defined');
}

for (let i in window.remark42) {
  if (i !== 'params') {
    components.push(i);
  }
}

for (let i = 0; i < components.length; i++) {
  const scripts =
    process.env.NODE_ENV === 'production'
      ? [createScript(params, components[i], false), createScript(params, components[i], true)]
      : [createScript(params, components[i], false)];
  scripts.map((s) => document.head.appendChild(s));
}

function createScript(params: RawConfigParams, name: string, isEsmodule: boolean) {
  const script = document.createElement('script');
  const ext = isEsmodule ? '.mjs' : '.js';

  if (isEsmodule) {
    script.type = 'module';
  } else {
    script.noModule = true;
  }

  script.async = true;
  script.defer = true;
  script.src = `${params.host}${name}${ext}`;

  return script;
}
