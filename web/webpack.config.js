const fs = require('fs');
const path = require('path');
const ExtractTextPlugin = require('extract-text-webpack-plugin');
const CleanPlugin = require('clean-webpack-plugin');
const ManifestPlugin = require('webpack-manifest-plugin');
const HtmlPlugin = require('html-webpack-plugin');
const webpack = require('webpack');

const publicFolder = path.resolve(__dirname, 'public');
const env = process.env.NODE_ENV || 'dev';
const hash = env === 'production' ? '.[chunkhash]' : '.[hash]';

const extractCSS = new ExtractTextPlugin({
  filename: `app${hash}.css`,
  allChunks: true
});
const cleanPublic = new CleanPlugin(publicFolder);
const uglifyJS = new webpack.optimize.UglifyJsPlugin();
const ModuleConcatenation = new webpack.optimize.ModuleConcatenationPlugin();
const Manifest = new ManifestPlugin();
const Html = new HtmlPlugin({ template: path.resolve(__dirname, 'index.ejs') })

const postcssLoader = {
  loader: 'postcss-loader',
  options: {
    plugins: [
      require('autoprefixer')({ browsers: ['> 1%'] }),
      require('postcss-url')({ url: 'inline', maxSize: 5 }),
      require('postcss-csso')
    ]
  }
};

module.exports = {
  context: __dirname,
  entry: {
    app: './app/app',
  },
  output: {
    path: publicFolder,
    // publicPath: '/path/to/public',
    filename: `app${hash}.js`
  },
  resolve: {
    modules: [ path.resolve(__dirname), 'node_modules' ]
  },
  module: {
    rules: [
      {
        test: /\.js$/,
        exclude: /(node_modules|\.vendor\.js$)/,
        use: {
          loader: 'babel-loader',
          options: {
            presets: [
              ['env', {
                targets: ['> 1%', 'android >= 4.4.4', 'ios >= 9'],
                useBuiltIns: true,
              }],
            ],
            plugins: ['transform-object-rest-spread'],
          }
        }
      },
      {
        test: /\.css$/,
        use: [ 'style-loader', 'css-loader', postcssLoader ]
        // TODO: style-loader must be turned on only for dev; for build we need extractCSS
      },
      {
        test: /\.scss$/,
        use: [
          'style-loader',
          'css-loader',
          postcssLoader,
          {
            loader: 'sass-loader',
            options: {
              // data: fs.readFileSync(path.resolve(__dirname, 'common/vars/vars.scss'), 'utf-8')
            }
          }
        ]
      },
      {
        test: /\.(png|jpg|jpeg|gif)$/,
        use: {
          loader: 'file-loader',
          options: {
            name: `files/[name].[hash].[ext]`
          }
        }
      }
    ]
  },
  plugins: [
    cleanPublic,
    Html, // we should add it only in dev
    extractCSS,
    // uglifyJS,
    ModuleConcatenation,
    Manifest,
  ],
  watch: env === 'dev',
  watchOptions: {
    ignored: /(node_modules|\.vendor\.js$)/
  },
  devServer: {
    host: '0.0.0.0',
    port: 8080,
    contentBase: publicFolder,
  },
};
