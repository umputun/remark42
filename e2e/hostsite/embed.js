// the loader snippet from the installation docs, unminified and shared by the pages here; each page
// sets remark_config before it includes this file
(function (c, d) {
  for (var i = 0; i < c.length; i++) {
    var s = d.createElement('script');
    s.type = 'module';
    s.async = true;
    s.defer = true;
    s.src = remark_config.host + '/web/' + c[i] + '.mjs';
    (d.head || d.body).appendChild(s);
  }
})(remark_config.components, document);
