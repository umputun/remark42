// The comment counter for host pages: every element with the class remark42__counter gets the number of
// comments on its page, the page named by the element's data-url, remark_config.url or the current
// location. A plain script served as written, with no module syntax, because the server answers the
// legacy counter.js name with these same bytes and an old integration loads that as a classic script.
(function () {
  'use strict';

  // the instance this script belongs to: remark_config.host when the page sets it, otherwise the address
  // the server fills in when it serves this file
  function baseUrl(host) {
    if (!host) {
      throw new Error("Remark42: remark_config.host wasn't configured.");
    }

    try {
      var protocol = new URL(host).protocol;

      // an http widget on an https page is blocked as mixed content, and an https one on an http page
      // works; neither is refused here, it is only reported
      if (protocol !== window.location.protocol) {
        console.error('Remark42: Protocol mismatch.');
      }

      // the host comes from the page, so anything but http and https is refused
      if (protocol !== 'http:' && protocol !== 'https:') {
        console.error('Remark42: Wrong protocol in host URL.');
        throw new Error();
      }
    } catch (e) {
      throw new Error('Remark42: Invalid host URL.');
    }

    return host;
  }

  function init() {
    var nodes = Array.prototype.slice.call(document.getElementsByClassName('remark42__counter'));
    var config = window.remark_config;

    if (!config) {
      throw new Error('Remark42: Config object is undefined.');
    }
    if (!config.site_id) {
      throw new Error('Remark42: Site ID is undefined.');
    }

    // the server fills the placeholder in when it serves this file; the frontend dev server serves it as
    // written, and there the page and the instance share an origin
    var instance = "{% REMARK_URL %}";
    if (instance.charAt(0) === '{') {
      instance = window.location.origin;
    }
    var base = baseUrl(config.host != null ? config.host : instance);
    var byUrl = {};

    nodes.forEach(function (node) {
      var url = node.dataset.url || config.url || window.location.origin + window.location.pathname;
      (byUrl[url] = byUrl[url] || []).push(node);
    });

    var request = new XMLHttpRequest();
    request.onreadystatechange = function () {
      if (request.readyState !== XMLHttpRequest.DONE || request.status !== 200) {
        return;
      }
      try {
        JSON.parse(request.responseText).forEach(function (item) {
          byUrl[item.url].forEach(function (node) {
            node.innerHTML = item.count.toString(10);
          });
        });
      } catch (e) {}
    };
    request.open('POST', base + '/api/v1/counts?site=' + config.site_id, true);
    request.setRequestHeader('Content-Type', 'application/json');
    request.send(JSON.stringify(Object.keys(byUrl)));
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
