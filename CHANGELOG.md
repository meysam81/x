# Changelog

## [1.12.0](https://github.com/meysam81/x/compare/v1.11.2...v1.12.0) (2025-08-31)


### Features

* add crypto lib ([4cd5aa3](https://github.com/meysam81/x/commit/4cd5aa3ee9cc64ffd17a97ff070667a0ec5b74c3))
* add smtp lib ([585d94c](https://github.com/meysam81/x/commit/585d94c9eb2b31572529160cc547b5ca5a898eaa))


### Bug Fixes

* **chimux:** modify typo ([5e09188](https://github.com/meysam81/x/commit/5e09188986ae22531e5c3abfd1ac605aad08882d))

## [1.11.2](https://github.com/meysam81/x/compare/v1.11.1...v1.11.2) (2025-08-09)


### Bug Fixes

* allow overriding tracer shutdown timeout ([e631e6e](https://github.com/meysam81/x/commit/e631e6e35dc88cfa8163143eb476c5eb52936e2a))

## [1.11.1](https://github.com/meysam81/x/compare/v1.11.0...v1.11.1) (2025-08-06)


### Bug Fixes

* **deps:** update module github.com/knadh/koanf/providers/env to v2 ([#33](https://github.com/meysam81/x/issues/33)) ([93b78ca](https://github.com/meysam81/x/commit/93b78ca49397c176fd51abf2899efd0f49363cf1))
* do not pass otlp endpoint if none is provided ([3360a94](https://github.com/meysam81/x/commit/3360a94fff445d2529de6ab2afd2fb513862a0d0))

## [1.11.0](https://github.com/meysam81/x/compare/v1.10.0...v1.11.0) (2025-08-06)


### Features

* add synthetic sleep test and benchmark test ([3646ec4](https://github.com/meysam81/x/commit/3646ec4c8cdfb5284e16d620e06f8dd83b2c98e9))
* add test for ratelimit ([f0b5173](https://github.com/meysam81/x/commit/f0b51736c81d5706491513172a6904d1086542ea))


### Bug Fixes

* close trace on context signal ([56735ec](https://github.com/meysam81/x/commit/56735ecc094a1007b8e50848686d544451c525ce))

## [1.10.0](https://github.com/meysam81/x/compare/v1.9.4...v1.10.0) (2025-07-12)


### Features

* add OTEL tracing lib ([fbf3b80](https://github.com/meysam81/x/commit/fbf3b805c5ec63c02291187777cafaa98e1e0487))


### Bug Fixes

* **deps:** update module github.com/knadh/koanf/providers/env to v2 ([#30](https://github.com/meysam81/x/issues/30)) ([fb5a707](https://github.com/meysam81/x/commit/fb5a7079bf27b22e1d471a8940bcf6a52c024333))

## [1.9.4](https://github.com/meysam81/x/compare/v1.9.3...v1.9.4) (2025-07-02)


### Bug Fixes

* **chimux:** disable logging middleware by default ([02a22ed](https://github.com/meysam81/x/commit/02a22edb14b2f1765a0478965781c8183af2c989))
* **chimux:** mast sensitive headers in logs ([b0310a3](https://github.com/meysam81/x/commit/b0310a390a99aec937f07c9fbd6a7111d12596f9))
* **deps:** update module github.com/redis/go-redis/v9 to v9.11.0 ([#27](https://github.com/meysam81/x/issues/27)) ([938c02f](https://github.com/meysam81/x/commit/938c02f441d53008ca6c4854927b622c029dcc89))
* **deps:** upgrade all libs ([2972918](https://github.com/meysam81/x/commit/29729184a718d04d87957c72eb30774bd9a5045a))
* disable metrics and healthz by default ([d26693a](https://github.com/meysam81/x/commit/d26693a206f2a709f29e8a39cc2ecea3b00cffc0))
* initialize log request with config ([c955ac4](https://github.com/meysam81/x/commit/c955ac4569a5ef02906b6c16973e50b160d7b2da))

## [1.9.3](https://github.com/meysam81/x/compare/v1.9.2...v1.9.3) (2025-06-26)


### Bug Fixes

* simplify log colors with enabled input ([a39087f](https://github.com/meysam81/x/commit/a39087f4d4796e0d213543949076c1b419e1a3eb))

## [1.9.2](https://github.com/meysam81/x/compare/v1.9.1...v1.9.2) (2025-06-23)


### Bug Fixes

* do not log metrics requests ([43539f6](https://github.com/meysam81/x/commit/43539f6231c72607327d156ea0ddc7a3194994a3))

## [1.9.1](https://github.com/meysam81/x/compare/v1.9.0...v1.9.1) (2025-06-22)


### Bug Fixes

* simplify log level argument ([ac6ca0a](https://github.com/meysam81/x/commit/ac6ca0aac8ce7dedef538a847be98f9a64bd6093))

## [1.9.0](https://github.com/meysam81/x/compare/v1.8.2...v1.9.0) (2025-06-22)


### Features

* add ratelimit lib ([d369d6e](https://github.com/meysam81/x/commit/d369d6ea128a225de14d542c90c3e39680a108f6))

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
