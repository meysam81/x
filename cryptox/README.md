# Crypto

```shell
$ go test -bench=. -benchmem ./crypto/...
goos: darwin
goarch: amd64
pkg: github.com/meysam81/x/crypto
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
```

## Bench 1

```plaintext
BenchmarkAESGCM_Encrypt_1KB-16                    459598              2316 ns/op         442.05 MB/s        2448 B/op          4 allocs/op
BenchmarkAESGCM_Decrypt_1KB-16                    734086              1404 ns/op         729.29 MB/s        2304 B/op          3 allocs/op
BenchmarkAESGCM_Encrypt_1MB-16                      1382            817338 ns/op        1282.92 MB/s     1058065 B/op          4 allocs/op
BenchmarkChaCha20_Encrypt_1KB-16                  640393              2124 ns/op         482.01 MB/s        1200 B/op          3 allocs/op
BenchmarkChaCha20_Decrypt_1KB-16                  925143              1340 ns/op         764.14 MB/s        1056 B/op          2 allocs/op
BenchmarkChaCha20_Encrypt_1MB-16                     888           1366304 ns/op         767.45 MB/s     1056819 B/op          3 allocs/op
BenchmarkArgon2id_Encrypt_Default-16                  18          62727618 ns/op        67121875 B/op         96 allocs/op
BenchmarkArgon2id_Decrypt_Default-16                  18          62887576 ns/op        67120728 B/op         93 allocs/op
BenchmarkArgon2id_Encrypt_HighSecurity-16              5         205374221 ns/op        134232331 B/op       136 allocs/op
BenchmarkPBKDF2_Encrypt_Default-16                     3         446586011 ns/op            4469 B/op         18 allocs/op
BenchmarkPBKDF2_Decrypt_Default-16                     3         455325727 ns/op            3162 B/op         15 allocs/op
BenchmarkPBKDF2_Encrypt_HighIterations-16              2         733501845 ns/op            4472 B/op         18 allocs/op
BenchmarkGenerateKey256-16                       1741222               686.2 ns/op            32 B/op          1 allocs/op
BenchmarkAESGCM_Parallel-16                       328702              3344 ns/op         306.24 MB/s        4752 B/op          7 allocs/op
BenchmarkChaCha20_Parallel-16                     527884              2545 ns/op         402.28 MB/s        2256 B/op          5 allocs/op
```

## Bench 2

```plaintext
BenchmarkAESGCM_Encrypt_1KB-16                    565636              2195 ns/op         466.52 MB/s        2432 B/op          3 allocs/op
BenchmarkAESGCM_Decrypt_1KB-16                    930876              1262 ns/op         811.72 MB/s        2304 B/op          3 allocs/op
BenchmarkAESGCM_Encrypt_1MB-16                      1694            857826 ns/op        1222.36 MB/s     1058049 B/op          3 allocs/op
BenchmarkChaCha20_Encrypt_1KB-16                  608810              1869 ns/op         547.94 MB/s        1184 B/op          2 allocs/op
BenchmarkChaCha20_Decrypt_1KB-16                  987297              1206 ns/op         848.94 MB/s        1056 B/op          2 allocs/op
BenchmarkChaCha20_Encrypt_1MB-16                     882           1493090 ns/op         702.29 MB/s     1056807 B/op          2 allocs/op
BenchmarkArgon2id_Encrypt_Default-16                  16          94655010 ns/op        67122092 B/op         95 allocs/op
BenchmarkArgon2id_Decrypt_Default-16                  18          63925862 ns/op        67121300 B/op         94 allocs/op
BenchmarkArgon2id_Encrypt_HighSecurity-16              5         210695180 ns/op        134232507 B/op       135 allocs/op
BenchmarkPBKDF2_Encrypt_Default-16                     3         398225141 ns/op            4453 B/op         17 allocs/op
BenchmarkPBKDF2_Decrypt_Default-16                     3         390489881 ns/op            3157 B/op         15 allocs/op
BenchmarkPBKDF2_Encrypt_HighIterations-16              2         654102038 ns/op            4456 B/op         17 allocs/op
BenchmarkGenerateKey256-16                       1932200               619.7 ns/op            32 B/op          1 allocs/op
BenchmarkAESGCM_Parallel-16                       434350              2920 ns/op         350.63 MB/s        4736 B/op          6 allocs/op
BenchmarkChaCha20_Parallel-16                     529720              2241 ns/op         456.91 MB/s        2240 B/op          4 allocs/op
```
