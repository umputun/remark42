require('dotenv').config();

const path = require('path');
const webpack = require('webpack');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const { CleanWebpackPlugin } = require('clean-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const TsconfigPathsPlugin = require('tsconfig-paths-webpack-plugin');
const ForkTsCheckerWebpackPlugin = require('fork-ts-checker-webpack-plugin');
const RefreshPlugin = require('@prefresh/webpack');
const { BundleAnalyzerPlugin } = require('webpack-bundle-analyzer');
const babelConfig = require('./.babelrc.js');

const NODE_ID = 'remark42';
const PUBLIC_PATH = '/web/';
const PORT = process.env.PORT || 9000;
const REMARK_API_BASE_URL = process.env.REMARK_API_BASE_URL || 'http://127.0.0.1:8080';
const DEVSERVER_BASE_PATH = process.env.DEVSERVER_BASE_PATH || 'http://127.0.0.1:9000';
const PUBLIC_FOLDER_PATH = path.resolve(__dirname, 'public');
const CUSTOM_PROPERTIES_PATH = path.resolve(__dirname, './app/custom-properties.css');

/**
 * Generates excludes for babel-loader
 *
 * Exclude is a module that has >=es6 code and resides in node_modules.
 * By defaut babel-loader ignores everything from node_modules,
 * so we have to exclude from ignore these modules
 */
const exclude = [
  '@github/markdown-toolbar-element',
  '@github/text-expander-element',
  'react-intl',
  'intl-messageformat',
  'intl-messageformat-parser',
].map((m) => path.resolve(__dirname, 'node_modules', m));

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
  // In development: We use `http://127.0.0.1:9000` for access to backend and backend is accessable via dev server proxy
  // In production: {% REMARK_URL %} will be replaced by `sed` on start of prod
  const REMARK_URL = isDev ? DEVSERVER_BASE_PATH : '{% REMARK_URL %}';

  // Add debug lib only for developmet throw webpack chuncks and keep code clear
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
    alias: {
      react: 'preact/compat',
      'react-dom': 'preact/compat',
    },
    plugins: [new TsconfigPathsPlugin()],
  };

  const output = {
    path: PUBLIC_FOLDER_PATH,
    publicPath: PUBLIC_PATH,
  };

  const getTsRule = (babelConfig = {}) => {
    return {
      test: /\.tsx?$/,
      exclude: /node_modules/,
      use: [
        {
          loader: 'babel-loader',
          options: {
            exclude,
            cacheDirectory: true,
            ...babelConfig,
          },
        },
        {
          loader: 'ts-loader',
          options: {
            transpileOnly: true,
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
              ['postcss-preset-env', { stage: 0 }],
              ['postcss-custom-properties', { importFrom: CUSTOM_PROPERTIES_PATH }],
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
          modules: {
            mode: 'local',
            localIdentName: '[name]__[local]_[hash:5]',
          },
        },
      },
      {
        loader: 'postcss-loader',
        options: {
          sourceMap: isDev,
          postcssOptions: {
            plugins: [
              ['postcss-preset-env', { stage: 0 }],
              ['postcss-custom-properties', { importFrom: CUSTOM_PROPERTIES_PATH }],
            ],
          },
        },
      },
    ],
  };

  const urlRule = {
    test: /\.(png|jpg|jpeg|gif|svg)$/,
    exclude: /node_modules/,
    use: {
      loader: 'url-loader',
      options: {
        name: '[name].[ext]',
        publicPath: PUBLIC_PATH,
        limit: 1200,
      },
    },
  };

  const rules = [cssRule, cssModulesRule, urlRule];

  const devServer = {
    port: PORT,
    contentBase: PUBLIC_FOLDER_PATH,
    disableHostCheck: true,
    hot: true,
    stats: 'minimal',
    watchOptions: {
      ignored: [PUBLIC_FOLDER_PATH, path.resolve(__dirname, 'node_modules')],
    },
    proxy: [
      { path: '/api', target: REMARK_API_BASE_URL, changeOrigin: true },
      { path: '/auth', target: REMARK_API_BASE_URL, changeOrigin: true },
    ],
  };

  const plugins = [
    ...(isDev ? [new CleanWebpackPlugin(), new webpack.HotModuleReplacementPlugin(), new RefreshPlugin()] : []),
    new webpack.DefinePlugin({
      'process.env.NODE_ENV': JSON.stringify(mode),
      'process.env.REMARK_NODE': JSON.stringify(NODE_ID),
      'process.env.REMARK_URL': isDev ? 'window.location.origin' : JSON.stringify(REMARK_URL),
    }),
    new MiniCssExtractPlugin({
      filename: '[name].css',
    }),
  ];

  const config = {
    entry,
    devtool: 'source-map',
    resolve,
  };

  const legacyConfig = {
    ...config,
    output: {
      ...output,
      filename: '[name].js',
      chunkFilename: '[name].js',
    },
    module: {
      rules: [getTsRule(), ...rules],
    },
    plugins: [
      ...plugins,
      ...(analyze
        ? [
            new BundleAnalyzerPlugin({
              analyzerMode: 'static',
              reportFilename: 'report-legacy.html',
              reportTitle: 'Legacy build',
            }),
          ]
        : []),
    ],
  };

  const modernConfig = {
    ...config,
    output: {
      ...output,
      filename: '[name].mjs',
      chunkFilename: '[name].mjs',
    },
    module: {
      rules: [
        getTsRule({
          ...babelConfig.env.modern,
          plugins: [...babelConfig.env.modern.plugins, ...(isDev ? ['@prefresh/babel-plugin'] : [])],
        }),
        ...rules,
      ],
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
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, 'templates/markdown-help.html'),
        filename: 'markdown-help.html',
        inject: false,
        minify: htmlMinifyOptions,
      }),
      new HtmlWebpackPlugin({
        template: path.resolve(__dirname, 'templates/privacy.html'),
        filename: 'privacy.html',
        inject: false,
        minify: htmlMinifyOptions,
      }),
      ...(analyze
        ? [
            new BundleAnalyzerPlugin({
              analyzerMode: 'static',
              reportFilename: 'report-modern.html',
              reportTitle: 'Modern build',
            }),
          ]
        : []),
    ],
    devServer,
  };

  if (isDev) {
    return modernConfig;
  }

  return [legacyConfig, modernConfig];
};

module.exports.CUSTOM_PROPERTIES_PATH = CUSTOM_PROPERTIES_PATH;
module.exports.exclude = exclude;
