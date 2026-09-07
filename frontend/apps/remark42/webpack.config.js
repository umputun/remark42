require('dotenv').config();

const path = require('path');
const webpack = require('webpack');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const { CleanWebpackPlugin } = require('clean-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const TsconfigPathsPlugin = require('tsconfig-paths-webpack-plugin');
const ForkTsCheckerWebpackPlugin = require('fork-ts-checker-webpack-plugin');
const { BundleAnalyzerPlugin } = require('webpack-bundle-analyzer');
const CssMinimizerPlugin = require('css-minimizer-webpack-plugin');
const incstr = require('incstr');

const NODE_ID = 'remark42';
const PUBLIC_PATH = '/web/';
const PORT = process.env.PORT || 9000;
const REMARK_API_BASE_URL = process.env.REMARK_API_BASE_URL || 'http://127.0.0.1:8080';
const DEVSERVER_BASE_PATH = process.env.DEVSERVER_BASE_PATH || `http://127.0.0.1:${PORT}`;
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

const htmlMinifyOptions = {
  minifyCSS: true,
  minifyJS: true,
  removeComments: true,
  removeRedundantAttributes: true,
  removeScriptTypeAttributes: true,
  removeStyleLinkTypeAttributes: true,
  sortAttributes: true,
  sortClassName: true,
  useShortDoctype: true,
};

module.exports = (_, { mode, analyze }) => {
  const isDev = mode === 'development';
  // Use REMARK_URL or predefined host in dev environment
  // In development: We use `http://127.0.0.1:9000` for access to backend and backend is accessible via dev server proxy
  // In production: {% REMARK_URL %} will be replaced by `sed` on start of prod
  const REMARK_URL = isDev ? DEVSERVER_BASE_PATH : '{% REMARK_URL %}';

  // Add debug lib only for development throw webpack chunks and keep code clear
  const preactDebug = isDev ? ['preact/debug'] : [];

  const entry = {
    embed: './app/embed.ts',
    counter: './app/counter.ts',
    deleteme: './app/deleteme.ts',
    'last-comments': [...preactDebug, CUSTOM_PROPERTIES_PATH, './app/last-comments.tsx'],
    remark: [...preactDebug, CUSTOM_PROPERTIES_PATH, './app/remark.tsx'],
  };

  const resolve = {
    extensions: ['.ts', '.tsx', '.js'],
    plugins: [new TsconfigPathsPlugin()],
  };

  const output = {
    path: PUBLIC_FOLDER_PATH,
    // derived from the url the bundle was loaded from rather than fixed to the domain root: an
    // instance mounted under a path, which manuals/separate-domain documents, would otherwise ask
    // for /web/<asset> and miss its own prefix
    publicPath: 'auto',
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
      isDev ? 'style-loader' : MiniCssExtractPlugin.loader,
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
      isDev ? 'style-loader' : MiniCssExtractPlugin.loader,
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
            plugins: [['postcss-preset-env', { stage: 0, features: { 'custom-properties': false } }]],
          },
        },
      },
    ],
  };

  const urlRule = {
    test: /\.(png|jpg|jpeg|gif|svg)$/,
    exclude: /node_modules/,
    use: {
      loader: 'file-loader',
      options: {
        // no publicPath here, so the emitted url goes through the runtime one above
        name: '[name].[ext]',
      },
    },
  };

  const rules = [cssRule, cssModulesRule, urlRule];

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
      // the bundler serves its own output from memory and wipes this directory on every dev build,
      // so watching it would only ever fire on the build's own writes
      { directory: PUBLIC_FOLDER_PATH, publicPath: PUBLIC_PATH, watch: false },
      // the assets the bundler does not build are served by the backend in production, so the dev
      // server reads them straight from where they live, or links to them 404 on this port
      { directory: WEB_ASSETS_PATH, publicPath: PUBLIC_PATH, watch: false },
    ],
    allowedHosts: 'all',
    hot: true,
    proxy: [
      { path: '/api', target: REMARK_API_BASE_URL, changeOrigin: true },
      { path: '/auth', target: REMARK_API_BASE_URL, changeOrigin: true },
    ],
    client: {
      overlay: false,
    },
  };

  const plugins = [
    new CleanWebpackPlugin(),
    new webpack.DefinePlugin({
      'process.env.NODE_ENV': JSON.stringify(mode),
      'process.env.REMARK_NODE': JSON.stringify(NODE_ID),
      'process.env.REMARK_URL': isDev ? 'window.location.origin' : JSON.stringify(REMARK_URL),
    }),
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
    plugins: [
      ...plugins,
      new ForkTsCheckerWebpackPlugin(),
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, 'templates/iframe.ejs'),
        filename: 'iframe.html',
        inject: false,
        env: mode,
        minify: htmlMinifyOptions,
      }),
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, 'templates/demo.ejs'),
        filename: 'index.html',
        inject: false,
        REMARK_URL,
        minify: htmlMinifyOptions,
      }),
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, 'templates/counter.ejs'),
        filename: 'counter.html',
        inject: false,
        REMARK_URL,
        minify: htmlMinifyOptions,
      }),
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, 'templates/last-comments.ejs'),
        filename: 'last-comments.html',
        inject: false,
        env: mode,
        REMARK_URL,
        minify: htmlMinifyOptions,
      }),
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, 'templates/deleteme.ejs'),
        filename: 'deleteme.html',
        inject: false,
        REMARK_URL,
        minify: htmlMinifyOptions,
      }),
      // the page the widget links to when third-party cookies are blocked, which is the only way
      // a reader in that state can reach the comments at all
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, 'templates/comments.ejs'),
        filename: 'comments.html',
        inject: false,
        REMARK_URL,
        minify: htmlMinifyOptions,
      }),
      ...(analyze
        ? [
            new BundleAnalyzerPlugin({
              analyzerMode: 'static',
              reportFilename: 'report.html',
              reportTitle: 'Bundle',
            }),
          ]
        : []),
    ],
    devServer,
  };

  return modernConfig;
};

module.exports.CUSTOM_PROPERTIES_PATH = CUSTOM_PROPERTIES_PATH;
