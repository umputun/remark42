# CHANGELOG

<a name="v1.0.5"></a>
## [v1.0.5] - 2026-11-18

### Bug Fixes

- Profiles that normalize, including SASLprep, now reject input whose
  decomposition has more than 30 consecutive combining characters. With
  SASLprep, such input produced output that changed when prepared again.
  This rejects some strings that v1.0.4 accepted. ([#7])

### Maintenance

- Change minimum Go version to 1.18
- Add a fuzz test for SASLprep idempotence

[#7]: https://github.com/xdg-go/stringprep/issues/7

<a name="v1.0.4"></a>
## [v1.0.4] - 2022-12-07

### Maintenance

- Bump golang.org/x/text to v0.3.8 due to CVE-2022-32149

<a name="v1.0.3"></a>
## [v1.0.3] - 2022-03-01

### Maintenance

- Bump golang.org/x/text to v0.3.7 due to CVE-2021-38561

<a name="v1.0.2"></a>
## [v1.0.2] - 2021-03-27

### Maintenance

- Change minimum Go version to 1.11

<a name="v1.0.1"></a>
## [v1.0.1] - 2021-03-24

### Bug Fixes

- Add go.mod file

<a name="v1.0.0"></a>
## [v1.0.0] - 2018-02-21

[v1.0.2]: https://github.com/xdg-go/stringprep/releases/tag/v1.0.2
[v1.0.1]: https://github.com/xdg-go/stringprep/releases/tag/v1.0.1
[v1.0.0]: https://github.com/xdg-go/stringprep/releases/tag/v1.0.0
