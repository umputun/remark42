# Reamark42 Website

## Work on Your Local Environment

Requirements:

- [Node.js v14](https://nodejs.org/en/) or higher - Install from package or with Homebrew
- Yarn 1.22 or higher - once you have Node.js run `npm i -g yarn`

### Development

Install dependencies and start development server:

```
$ yarn
$ yarn dev
```

### Build

```
$ yarn build
```

## Work with Docker Compose

### Build

Install dependencies and run development server inside Docker:

```
$ docker-compose build
$ docker-compose up server
```

Then serve files from `./build` with your favorite server

### Development

```
$ docker-compose up --build server
```

Then head to http://localhost:8080
