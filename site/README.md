# Remark42 site

## Work on your local environment

Requirements:

* [Node.js v14](https://nodejs.org/en/) or higher - install from package or with Homebrew
* Yarn 1.22 or higher - once you have Node.js, run `npm i -g yarn`

### Development

Install dependencies and start the development server:

```shell
yarn
yarn dev
```

### Build

```shell
yarn build
```

## Work with Docker Compose

### Build

Install dependencies and run development server inside Docker:

```shell
docker-compose build
docker-compose up server
```

Then serve files from `./build` with your favorite server

### Development

```shell
docker-compose up --build server
```

### New docker compose file

```shell
docker-compose -f docker-compose-build.yml up --build build
docker-compose -f docker-compose-build.yml up --build livereload
```

Then head to http://localhost:8080
