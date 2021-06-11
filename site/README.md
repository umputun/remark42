# Reamark42 Website

## Build site with Docker

```
$ docker-compose build
$ docker-compose run build
```

Then serve files from `./build` with your favorite server

## Development

### With docker

```
$ docker-compose build
$ docker-compose up server
```

Then head to http://127.0.0.1:8080

### Without docker

Requirements:

- Node.js v14 or higher
- Yarn 1.22 or higher

Install dependencies:

```
$ yarn
```

Run dev server with hot reload:

```
$ yarn dev
```
