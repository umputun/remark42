const path = require('path');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const CssMinimizerPlugin = require('css-minimizer-webpack-plugin');
const incstr = require('incstr');

// a local .env may override the variables read below, PORT among them. loadEnvFile throws when
// the file is absent, which is the usual case
try {
  process.loadEnvFile(path.resolve(__dirname, '.env'));
} catch (e) {
  if (e.code !== 'ENOENT') {
    throw e;
  }
}

const PUBLIC_PATH = '/web/';
const PORT = process.env.PORT || 9000;
const REMARK_API_BASE_URL = process.env.REMARK_API_BASE_URL || 'http://127.0.0.1:8080';
const PUBLIC_FOLDER_PATH = path.resolve(__dirname, 'public');
const WEB_ASSETS_PATH = path.resolve(__dirname, '../../../backend/app/webassets/assets');
const CUSTOM_PROPERTIES_PATH = path.resolve(__dirname, './app/styles/custom-properties.css');

const genId = incstr.idGenerator();
const modulesMap = {};

function getLocalIdent(loaderContext, _, localName, options) {
  if (!options.context) {
    options.context = loaderContext.rootContext;
  }

  const filepath = path.relative(options.context, loaderContext.resourcePath).replace(/\\/g, '/');

  if (!modulesMap[filepath]) {
    modulesMap[filepath] = { id: genId(), genId: incstr.idGenerator(), classNames: {} };
  }

  const m = modulesMap[filepath];

  if (!m.classNames[localName]) {
    m.classNames[localName] = m.genId();
  }

  return `${m.id}_${m.classNames[localName]}`;
}

module.exports = (env, { mode }) => {
  const isDev = mode === 'development';
  // `webpack serve --env apiBase=<url>` points the dev server's proxy at another backend
  const apiBase = env.apiBase || REMARK_API_BASE_URL;
  // Add debug lib only for development throw webpack chunks and keep code clear
  const preactDebug = isDev ? ['preact/debug'] : [];

  const entry = {
    embed: './app/embed.ts',
    deleteme: './app/deleteme.ts',
    'last-comments': [...preactDebug, CUSTOM_PROPERTIES_PATH, './app/last-comments.tsx'],
    remark: [...preactDebug, CUSTOM_PROPERTIES_PATH, './app/remark.tsx'],
  };

  const resolve = {
    extensions: ['.ts', '.tsx', '.js'],
    // imports are written from the app root (`common/…`, `components/…`). The app directory comes
    // first, the order baseUrl gives the type checker in tsconfig.json
    modules: [path.resolve(__dirname, 'app'), 'node_modules'],
  };

  const output = {
    path: PUBLIC_FOLDER_PATH,
    // derived from the url the bundle was loaded from rather than fixed to the domain root: an
    // instance mounted under a path, which manuals/separate-domain documents, would otherwise ask
    // for /web/<asset> and miss its own prefix
    publicPath: 'auto',
    clean: true,
  };

  const getTsRule = () => {
    return {
      test: /\.tsx?$/,
      exclude: /node_modules/,
      use: [
        {
          loader: 'babel-loader',
          options: {
            cacheDirectory: true,
          },
        },
      ],
    };
  };

  const cssRule = {
    test: /\.css$/,
    exclude: [/\.module\.css$/, /node_modules/],
    use: [
      MiniCssExtractPlugin.loader,
      'css-loader',
      {
        loader: 'postcss-loader',
        options: {
          sourceMap: isDev,
          postcssOptions: {
            plugins: [
              [
                'postcss-preset-env',
                {
                  browsers: 'defaults, not IE 11, not samsung 12',
                  stage: 0,
                  features: {
                    'custom-properties': false,
                    // ships logical properties (margin-inline-start, etc.) as authored instead of
                    // generating [dir=ltr]/[dir=rtl] selector pairs: those pairs key off the page's
                    // html[dir] ancestor rather than an element's own resolved direction, which
                    // mis-mirrors content whose direction differs from the page (e.g. an LTR reply
                    // inside an RTL thread). Native logical properties follow each element's own
                    // direction correctly; the cost is Opera Mini, KaiOS 2.5 and QQ 14.9 (~0.1%
                    // global usage) not resolving them
                    'logical-properties-and-values': false,
                  },
                },
              ],
            ],
          },
        },
      },
    ],
  };

  const cssModulesRule = {
    test: /\.module\.css$/,
    exclude: /node_modules/,
    use: [
      MiniCssExtractPlugin.loader,
      {
        loader: 'css-loader',
        options: {
          importLoaders: 1,
          modules: {
            localIdentName: '[name]__[local]_[hash:5]',
            getLocalIdent: isDev ? undefined : getLocalIdent,
            // css-loader 7 defaults CSS modules to named exports; the components import the
            // whole map as a default, and several class names are not valid identifiers
            namedExport: false,
            exportLocalsConvention: 'as-is',
          },
        },
      },
      {
        loader: 'postcss-loader',
        options: {
          sourceMap: isDev,
          postcssOptions: {
            plugins: [
              [
                'postcss-preset-env',
                // see the matching features comment in cssRule above
                { stage: 0, features: { 'custom-properties': false, 'logical-properties-and-values': false } },
              ],
            ],
          },
        },
      },
    ],
  };

  const rules = [cssRule, cssModulesRule];

  const devServer = {
    port: PORT,
    devMiddleware: {
      stats: 'minimal',
      // serve the in-memory build output under the same PUBLIC_PATH the devServer.static entries
      // and every template's script tags use. output.publicPath stays 'auto' for the runtime
      // bundle; this only affects where the dev server itself serves its own output from.
      publicPath: PUBLIC_PATH,
    },
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, PATCH, OPTIONS',
      'Access-Control-Allow-Headers': 'X-Requested-With, content-type, Authorization',
    },
    static: [
      // entries are consulted in order, so the build output comes first here for the same reason
      // it does in the backend's file server
      // the bundler serves its own output from memory and answers before this entry does, so
      // nothing here changes while the server runs and there is nothing to watch
      { directory: PUBLIC_FOLDER_PATH, publicPath: PUBLIC_PATH, watch: false },
      // the assets the bundler does not build are served by the backend in production, so the dev
      // server reads them straight from where they live, or links to them 404 on this port
      { directory: WEB_ASSETS_PATH, publicPath: PUBLIC_PATH, watch: false },
    ],
    allowedHosts: 'all',
    hot: true,
    proxy: [
      { path: '/api', target: apiBase, changeOrigin: true },
      { path: '/auth', target: apiBase, changeOrigin: true },
    ],
    client: {
      overlay: false,
    },
  };

  const plugins = [
    new MiniCssExtractPlugin({
      filename: '[name].css',
    }),
  ];

  const optimization = {
    // doc: https://webpack.js.org/plugins/css-minimizer-webpack-plugin/
    minimizer: [`...`, new CssMinimizerPlugin()],
  };

  const config = {
    entry,
    devtool: isDev ? 'source-map' : false,
    resolve,
    optimization,
  };

  const modernConfig = {
    ...config,
    output: {
      ...output,
      filename: '[name].mjs',
      chunkFilename: '[name].mjs',
    },
    module: {
      rules: [getTsRule(), ...rules],
    },
    plugins,
    devServer,
  };

  return modernConfig;
};

module.exports.CUSTOM_PROPERTIES_PATH = CUSTOM_PROPERTIES_PATH;
