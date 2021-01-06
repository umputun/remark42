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
const REMARK_URL = '{% REMARK_URL %}';
const PUBLIC_PATH = '/web';
const PUBLIC_FOLDER_PATH = path.resolve(__dirname, 'public');
const CUSTOM_PROPERTIES_PATH = path.resolve(__dirname, './app/custom-properties.css');

const exclude = [
  '@github/markdown-toolbar-element',
  '@github/text-expander-element',
  'react-intl',
  'intl-messageformat',
  'intl-messageformat-parser',
].map(m => path.resolve(__dirname, 'node_modules', m));

module.exports = (_, { mode, analyze, env }) => {
  const isDev = mode === 'development';

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

  const optimization = {
    chunkIds: 'named',
    moduleIds: 'named',
    splitChunks: {
      minChunks: 3,
    },
  };

  const getTsRule = (babelEnvConfig = {}) => {
    return {
      test: /\.tsx?$/,
      use: [
        {
          loader: 'babel-loader',
          options: {
            cacheDirectory: true,
            ...babelEnvConfig,
          },
        },
        {
          loader: 'ts-loader',
          options: {
            transpileOnly: true,
          },
        },
      ],
      /**
       * Generates excludes for babel-loader
       *
       * Exclude is a module that has >=es6 code and resides in node_modules.
       * By defaut babel-loader ignores everything from node_modules,
       * so we have to exclude from ignore these modules
       */
      exclude,
    };
  };

  const cssRule = {
    test: /\.css$/,
    exclude: /\.module\.css$/,
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

  const fileRule = {
    test: /\.(png|jpg|jpeg|gif|svg)$/,
    use: {
      loader: 'file-loader',
      options: {
        name: isDev ? 'files/[name].[contenthash].[ext]' : 'files/[contenthash:6].[ext]',
      },
    },
  };

  const rules = [cssRule, cssModulesRule, fileRule];

  const plugins = [
    ...(isDev
      ? [
          new CleanWebpackPlugin(),
          new RefreshPlugin(),
          new webpack.ProgressPlugin(),
          new webpack.HotModuleReplacementPlugin(),
        ]
      : []),
    new webpack.DefinePlugin({
      'process.env.NODE_ENV': JSON.stringify(mode),
      'process.env.REMARK_NODE': JSON.stringify(NODE_ID),
      'process.env.REMARK_URL': isDev ? 'window.location.origin' : JSON.stringify(REMARK_URL),
    }),
    new MiniCssExtractPlugin({
      filename: '[name].css',
    }),
    new ForkTsCheckerWebpackPlugin(),
    new HtmlWebpackPlugin({
      template: path.resolve(__dirname, 'templates/iframe.ejs'),
      filename: 'iframe.html',
      inject: false,
      env: mode,
    }),
    new HtmlWebpackPlugin({
      template: path.resolve(__dirname, 'templates/demo.ejs'),
      filename: 'index.html',
      inject: false,
      REMARK_URL,
    }),
    new HtmlWebpackPlugin({
      template: path.resolve(__dirname, 'templates/counter.ejs'),
      filename: 'counter.html',
      inject: false,
      REMARK_URL,
    }),
    new HtmlWebpackPlugin({
      template: path.resolve(__dirname, 'templates/last-comments.ejs'),
      filename: 'last-comments.html',
      inject: false,
      REMARK_URL,
    }),
    new HtmlWebpackPlugin({
      template: path.resolve(__dirname, 'templates/deleteme.ejs'),
      filename: 'deleteme.html',
      inject: false,
      REMARK_URL,
    }),
    new HtmlWebpackPlugin({
      template: path.resolve(__dirname, 'templates/markdown-help.html'),
      filename: 'markdown-help.html',
      inject: false,
    }),
    new HtmlWebpackPlugin({
      template: path.resolve(__dirname, 'templates/privacy.html'),
      filename: 'privacy.html',
      inject: false,
    }),
  ];

  const devServer = {
    host: '0.0.0.0',
    port: process.env.PORT || 9000,
    contentBase: PUBLIC_FOLDER_PATH,
    publicPath: PUBLIC_PATH,
    disableHostCheck: true,
    historyApiFallback: true,
    quiet: true,
    inline: true,
    hot: true,
    compress: true,
    clientLogLevel: 'none',
    overlay: false,
    stats: 'minimal',
    watchOptions: {
      ignored: [path.resolve(__dirname, 'build'), path.resolve(__dirname, 'node_modules')],
    },
    proxy: [
      { path: '/api', target: REMARK_URL, changeOrigin: true },
      { path: '/auth', target: REMARK_URL, changeOrigin: true },
    ],
  };

  const config = {
    entry,
    resolve,
    optimization,
    stats: 'minimal',
    devServer: env.WEBPACK_SERVE && devServer,
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
      rules: [getTsRule(babelConfig.env.modern), ...rules],
    },
    plugins: [
      ...plugins,
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
  };

  if (isDev) {
    return modernConfig;
  }

  return [legacyConfig, modernConfig];
};
module.exports.CUSTOM_PROPERTIES_PATH = CUSTOM_PROPERTIES_PATH;
module.exports.exclude = exclude;
