# Changelog

## [1.8.2](https://github.com/meysam81/x/compare/v1.8.1...v1.8.2) (2025-06-15)


### Bug Fixes

* **logging:** pass timeformat to the console writer ([a3a879a](https://github.com/meysam81/x/commit/a3a879a15c7a2bddb018b33a876016e080d3cf4f))

## [1.8.1](https://github.com/meysam81/x/compare/v1.8.0...v1.8.1) (2025-06-04)


### Bug Fixes

* **logging:** return non-pointer logger with caller attached ([2dc2f1d](https://github.com/meysam81/x/commit/2dc2f1d21f64dbd2178815b0d528790528e38fad))

## [1.8.0](https://github.com/meysam81/x/compare/v1.7.1...v1.8.0) (2025-06-03)


### Features

* **chimux:** disable healthcheck logging by default ([98c1d06](https://github.com/meysam81/x/commit/98c1d06b27c2a6b7941afbfb9c36c1f5e2a60549))

## [1.7.1](https://github.com/meysam81/x/compare/v1.7.0...v1.7.1) (2025-05-31)


### Bug Fixes

* **chimux:** lower header key in logging ([b1eb30f](https://github.com/meysam81/x/commit/b1eb30f0323cfc4dc6cc81a37839c673d8aeebd0))

## [1.7.0](https://github.com/meysam81/x/compare/v1.6.0...v1.7.0) (2025-05-31)


### Features

* **chimux:** provide health and metrics endpoint by default ([65b6e85](https://github.com/meysam81/x/commit/65b6e85aa0dd10f608c078bcad7a9cdd32dcf981))

## [1.6.0](https://github.com/meysam81/x/compare/v1.5.0...v1.6.0) (2025-05-31)


### Features

* **config:** allow unmarshaling to user-provided interface ([9dadf61](https://github.com/meysam81/x/commit/9dadf6183485127d63c29d2fd10013071a382d29))


### Bug Fixes

* remove deprecated chi version ([66d93aa](https://github.com/meysam81/x/commit/66d93aa239df3fe15f9fe1fa0d683bd97cffe1cc))

## [1.5.0](https://github.com/meysam81/x/compare/v1.4.0...v1.5.0) (2025-05-31)


### Features

* **chimux:** provide chi router with logging and recovery middleware ([c4dfd68](https://github.com/meysam81/x/commit/c4dfd680ca1d8503ff52570c0f3720c48dcada38))
* **config:** allow providing defaults in constructor ([14dd775](https://github.com/meysam81/x/commit/14dd775e360c85d39c589be7eaa38796e2154207))
* **logging:** return a pointer to the logger instead ([b663d33](https://github.com/meysam81/x/commit/b663d33489f7f84bff1dc8c024a89c43504da1bb))


### Bug Fixes

* **config:** comply to the api and overridden options ([951b99b](https://github.com/meysam81/x/commit/951b99ba13f7659b354d5eb880c79ae49921c1a8))
* **gin:** make linter happy ([bde3c78](https://github.com/meysam81/x/commit/bde3c784c1468f78c57465e5ce3a2396abe7c2f9))

## [1.4.0](https://github.com/meysam81/x/compare/v1.3.1...v1.4.0) (2025-05-30)


### Features

* **logging:** add http middleware ([d4718a6](https://github.com/meysam81/x/commit/d4718a68a5d575ae8b240a72287b775241c9b53c))

## [1.3.1](https://github.com/meysam81/x/compare/v1.3.0...v1.3.1) (2025-05-25)


### Bug Fixes

* **config:** add dash handling to env loader ([0f641d8](https://github.com/meysam81/x/commit/0f641d88aa8439ee603e5d30c9dbceb68dffccb1))

## [1.3.0](https://github.com/meysam81/x/compare/v1.2.1...v1.3.0) (2025-05-24)


### Features

* **config:** allow customizing the env prefix ([f7a3a9c](https://github.com/meysam81/x/commit/f7a3a9c410f324cd8106f9c3ce31512c6310e459))

## [1.2.1](https://github.com/meysam81/x/compare/v1.2.0...v1.2.1) (2025-05-24)


### Bug Fixes

* move zerolog logger to gin options ([d4a6f4b](https://github.com/meysam81/x/commit/d4a6f4bb4bb697e670b2dc3408c76bc1045969e8))

## [1.2.0](https://github.com/meysam81/x/compare/v1.1.0...v1.2.0) (2025-05-23)


### Features

* allow preload the config with default first ([49aa69d](https://github.com/meysam81/x/commit/49aa69d81b699c5e417564e8489676dd6f4731ef))


### Bug Fixes

* rename files to comply with go community ([593eb2d](https://github.com/meysam81/x/commit/593eb2d652be45f6b87d461ea3c9ea1cad8902ea))
* rename the http logging module ([dc6ab26](https://github.com/meysam81/x/commit/dc6ab266cb500cd4413cbeab6eee518f7bf9fce8))

## [1.1.0](https://github.com/meysam81/x/compare/v1.0.1...v1.1.0) (2025-05-23)


### Features

* add gin lib ([7a2a765](https://github.com/meysam81/x/commit/7a2a76553cbf623670e9bed3e5189b863e85e1c1))
* add logging with zerolog ([bfc7c8b](https://github.com/meysam81/x/commit/bfc7c8bc2838ff5ba8154e44053029a833f86dfb))
* add nethttp-logging middleware ([e57f539](https://github.com/meysam81/x/commit/e57f5397858b579cafcef7a7eeea3b09727acfc5))

## [1.0.1](https://github.com/meysam81/x/compare/v1.0.0...v1.0.1) (2025-05-22)


### Bug Fixes

* modify blank type syntax ([232150c](https://github.com/meysam81/x/commit/232150c05c1cd6cfb31feb76b7073b05a304b96f))

## 1.0.0 (2025-05-22)


### Features

* add koanf config ([30a6898](https://github.com/meysam81/x/commit/30a6898aac14b16c8c76ef70dc81e902d12112e4))
* **CI:** add release please ([4851000](https://github.com/meysam81/x/commit/4851000bc4bf04d7211c87c7cb20a25bb50cf636))
* initial commit ([078b9ab](https://github.com/meysam81/x/commit/078b9ab1cd30890e43506636f13b7e63b93a9054))


### Bug Fixes

* allow cgo disabled builds for sqlite ([cc1230e](https://github.com/meysam81/x/commit/cc1230eac50b666704e387bc37ca66ccaee2a1c7))
* modify the engine per cgo build tag ([c30ffe5](https://github.com/meysam81/x/commit/c30ffe52a07fb87b69fcb62998488855b9d369f2))
