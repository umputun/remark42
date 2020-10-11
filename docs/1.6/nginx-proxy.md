---
title: Nginx
---

## How to configure remark42 with nginx reverse proxy

Example of nginx configuration (reverse proxy) running remark42 service on remark42.example.com

```
server {
    listen      443;
    server_name remark42.example.com;
    ssl    on;
    ssl_certificate        /etc/nginx/ssl/remark42.example.com.crt;
    ssl_certificate_key    /etc/nginx/ssl/remark42.example.com.key;

    gzip on;
    gzip_types text/plain application/json text/css application/javascript application/x-javascript text/javascript text/xml application/xml application/rss+xml application/atom+xml application/rdf+xml;
    gzip_min_length 1000;
    gzip_proxied any;


    location ~ /\.git {
        deny all;
    }

    location /index.html {
         proxy_redirect          off;
         proxy_set_header        X-Real-IP $remote_addr;
         proxy_set_header        X-Forwarded-For $proxy_add_x_forwarded_for;
         proxy_set_header        Host $http_host;
         proxy_pass              http://remark42:8080/web/index.html;
     }

    location / {
         proxy_redirect          off;
         proxy_set_header        X-Real-IP $remote_addr;
         proxy_set_header        X-Forwarded-For $proxy_add_x_forwarded_for;
         proxy_set_header        Host $http_host;
         proxy_pass              http://remark42:8080/;
    }

    access_log   /var/log/nginx/remark42.log;

}

server {
  listen 80;
  server_name remark42.example.com;
  return      301 https://remark42.example.com$request_uri;
}
```

note: `proxy_pass` points to internal DNS name `remark42` and expected to run from the same compose. If nginx runs outside of compose the real IP (or docker's bridge IP) should be used
