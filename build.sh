#!/bin/sh -e
pluginname="anydesk"

build() {
	GOOS="$1"
	GOARCH="$2"
	
	namesuffix="_${GOOS}_${GOARCH}"

	case "$GOOS" in
	windows)
		suffix=.dll
		CGO_ENABLED=1
		case "$GOARCH" in
		amd64)
			namesuffix="_win64"
			CC=x86_64-w64-mingw32-gcc
			;;
		386)
			namesuffix="_win32"
			CC=i686-w64-mingw32-gcc
			;;
		esac
		;;
	darwin)
		suffix=.dylib
		namesuffix="_mac"
		CC=x86_64-apple-darwin20.2-cc
		CGO_ENABLED=1
		PATH="/opt/osxcross/bin:$PATH"
		LD_LIBRARY_PATH="/opt/osxcross/lib"
		;;
	linux)
		suffix=.so
		CGO_ENABLED=1
		CC=x86_64-pc-linux-gnu-gcc
		case "$GOARCH" in
		amd64)
			namesuffix="_linux_amd64"
			;;
		386)
			namesuffix="_linux_x86"
			CFLAGS=-m32
			;;
		esac
		;;
	esac

	output="plugins/${pluginname}${namesuffix}${suffix}"

	echo "## Building for $GOOS/$GOARCH => $output" >&2
	echo "" >&2

	mkdir -vp "$(dirname "${output}")"
	PATH="$PATH" CFLAGS="$CFLAGS" CC="$CC" LD_LIBRARY_PATH="$LD_LIBRARY_PATH" \
	GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED="$CGO_ENABLED" go build -v -buildmode=c-shared -ldflags="-s -w" -o "$output"
	echo "" >&2
}

package() {
	rm -vf "${pluginname}.ts3_plugin"
	zip -9 "${pluginname}.ts3_plugin" package.ini plugins/*
}

build windows 386
build windows amd64
build linux 386
build linux amd64
build darwin amd64

package
