# Changelog

## [1.12.2](https://github.com/meysam81/x/compare/chimux-v1.0.0...chimux-v1.12.2) (2026-02-19)


### Features

* **chimux:** disable healthcheck logging by default ([98c1d06](https://github.com/meysam81/x/commit/98c1d06b27c2a6b7941afbfb9c36c1f5e2a60549))
* **chimux:** provide chi router with logging and recovery middleware ([c4dfd68](https://github.com/meysam81/x/commit/c4dfd680ca1d8503ff52570c0f3720c48dcada38))
* **chimux:** provide health and metrics endpoint by default ([65b6e85](https://github.com/meysam81/x/commit/65b6e85aa0dd10f608c078bcad7a9cdd32dcf981))
* create monorepo per-dir module for reduced dependency hell ([7072769](https://github.com/meysam81/x/commit/7072769bfc890c146b7caf12af6edc8d0f0f31c4))


### Bug Fixes

* **chimux:** disable logging middleware by default ([02a22ed](https://github.com/meysam81/x/commit/02a22edb14b2f1765a0478965781c8183af2c989))
* **chimux:** lower header key in logging ([b1eb30f](https://github.com/meysam81/x/commit/b1eb30f0323cfc4dc6cc81a37839c673d8aeebd0))
* **chimux:** mast sensitive headers in logs ([b0310a3](https://github.com/meysam81/x/commit/b0310a390a99aec937f07c9fbd6a7111d12596f9))
* **chimux:** modify typo ([5e09188](https://github.com/meysam81/x/commit/5e09188986ae22531e5c3abfd1ac605aad08882d))
* **CI:** release 1.12.2 ([3d24fcb](https://github.com/meysam81/x/commit/3d24fcb67c2d58a6b75a00f36d4742c49c39a60c))
* disable metrics and healthz by default ([d26693a](https://github.com/meysam81/x/commit/d26693a206f2a709f29e8a39cc2ecea3b00cffc0))
* do not log metrics requests ([43539f6](https://github.com/meysam81/x/commit/43539f6231c72607327d156ea0ddc7a3194994a3))
* initialize log request with config ([c955ac4](https://github.com/meysam81/x/commit/c955ac4569a5ef02906b6c16973e50b160d7b2da))
* **logging:** return non-pointer logger with caller attached ([2dc2f1d](https://github.com/meysam81/x/commit/2dc2f1d21f64dbd2178815b0d528790528e38fad))
* remove deprecated chi version ([66d93aa](https://github.com/meysam81/x/commit/66d93aa239df3fe15f9fe1fa0d683bd97cffe1cc))
