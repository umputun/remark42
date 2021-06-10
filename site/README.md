# Reamark42 Website

## Build site with Docker

```
$ docker build -t remark42-site .
$ docker cp remark42-site:/site ./path/to/folder
```

Than serve files with your favorite server

## Development

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
