test-crypto-elg-all: test-crypto-elg test-crypto-elg-benchmarks

test-crypto-elg:
	go test -v ./lib/crypto -run TestElg

test-crypto-elg-benchmarks:
	go test -v ./lib/crypto -bench=Elg -run=^$

# Individual benchmarks
test-crypto-elg-bench-generate:
	go test -v ./lib/crypto -bench=ElgGenerate -run=^$

test-crypto-elg-bench-encrypt:
	go test -v ./lib/crypto -bench=ElgEncrypt -run=^$

test-crypto-elg-bench-decrypt:
	go test -v ./lib/crypto -bench=ElgDecrypt -run=^$

.PHONY: test-crypto-elg-all \
        test-crypto-elg \
        test-crypto-elg-benchmarks \
        test-crypto-elg-bench-generate \
        test-crypto-elg-bench-encrypt \
        test-crypto-elg-bench-decrypt