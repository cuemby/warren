.PHONY: build build-optimized compress report clean

# Build with default settings
build:
	@echo "Building with default settings..."
	go build -o warren-default .
	@ls -lh warren-default
	@echo ""

# Build with optimizations
build-optimized:
	@echo "Building with optimizations (-ldflags=\"-s -w\")..."
	CGO_ENABLED=0 go build -ldflags="-s -w" -o warren-optimized .
	@ls -lh warren-optimized
	@echo ""

# Compress with UPX
compress: build-optimized
	@echo "Compressing with UPX..."
	@if command -v upx >/dev/null 2>&1; then \
		cp warren-optimized warren-compressed; \
		upx --best --lzma warren-compressed; \
		ls -lh warren-compressed; \
	else \
		echo "UPX not installed. Install with:"; \
		echo "  macOS: brew install upx"; \
		echo "  Linux: sudo apt install upx"; \
	fi
	@echo ""

# Generate size report
report: build build-optimized
	@echo "=== Binary Size Report ===" > size-report.txt
	@echo "" >> size-report.txt
	@echo "Default build:" >> size-report.txt
	@ls -lh warren-default | awk '{print "  Size: " $$5}' >> size-report.txt
	@echo "" >> size-report.txt
	@echo "Optimized build (-ldflags=\"-s -w\"):" >> size-report.txt
	@ls -lh warren-optimized | awk '{print "  Size: " $$5}' >> size-report.txt
	@echo "" >> size-report.txt
	@if [ -f warren-compressed ]; then \
		echo "Compressed (UPX):" >> size-report.txt; \
		ls -lh warren-compressed | awk '{print "  Size: " $$5}' >> size-report.txt; \
		echo "" >> size-report.txt; \
	fi
	@echo "Report saved to size-report.txt"
	@cat size-report.txt

# Clean build artifacts
clean:
	rm -f warren-default warren-optimized warren-compressed size-report.txt

# Full test
all: build build-optimized compress report
