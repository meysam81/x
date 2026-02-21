# Changelog

## [1.12.4](https://github.com/meysam81/x/compare/tracing-v1.12.3...tracing-v1.12.4) (2026-02-21)


### Bug Fixes

* remove replace for dependencies to x/logging ([73e0edc](https://github.com/meysam81/x/commit/73e0edc2fb92bd7a69b706443864d7a4afa33760))

## [1.12.3](https://github.com/meysam81/x/compare/tracing-v1.12.2...tracing-v1.12.3) (2026-02-19)


### Bug Fixes

* **deps:** update module github.com/go-chi/chi/v5 to v5.2.5 ([#45](https://github.com/meysam81/x/issues/45)) ([19069ee](https://github.com/meysam81/x/commit/19069eefc07f43ae78f07e3a4100b679042b343b))
* **deps:** update opentelemetry-go monorepo to v1.40.0 ([#48](https://github.com/meysam81/x/issues/48)) ([5317ea4](https://github.com/meysam81/x/commit/5317ea4ed06ad61812497001e3aa22171ce09135))

## [1.12.2](https://github.com/meysam81/x/compare/tracing-v1.0.0...tracing-v1.12.2) (2026-02-19)


### Features

* add OTEL tracing lib ([fbf3b80](https://github.com/meysam81/x/commit/fbf3b805c5ec63c02291187777cafaa98e1e0487))
* create monorepo per-dir module for reduced dependency hell ([7072769](https://github.com/meysam81/x/commit/7072769bfc890c146b7caf12af6edc8d0f0f31c4))


### Bug Fixes

* allow overriding tracer shutdown timeout ([e631e6e](https://github.com/meysam81/x/commit/e631e6e35dc88cfa8163143eb476c5eb52936e2a))
* **CI:** release 1.12.2 ([3d24fcb](https://github.com/meysam81/x/commit/3d24fcb67c2d58a6b75a00f36d4742c49c39a60c))
* close trace on context signal ([56735ec](https://github.com/meysam81/x/commit/56735ecc094a1007b8e50848686d544451c525ce))
* do not pass otlp endpoint if none is provided ([3360a94](https://github.com/meysam81/x/commit/3360a94fff445d2529de6ab2afd2fb513862a0d0))
