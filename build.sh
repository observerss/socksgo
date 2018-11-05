#!/usr/bin/env bash
mkdir -p dist/
rm dist/*
VERSION=0.1
GOEXEC=/usr/local/bin/go
GOMOBILE=/Users/ob/go/pkg/gomobile
GOMOBILECC=$GOMOBILE/ndk-toolchains/arm/bin/arm-linux-androideabi-gcc
GOOS=windows GOARCH=386 $GOEXEC build -o dist/socksgo-windows-386-$VERSION.exe
GOOS=windows GOARCH=amd64 $GOEXEC build -o dist/socksgo-windows-amd64-$VERSION.exe
GOOS=darwin GOARCH=amd64 $GOEXEC build -o dist/socksgo-darwin-amd64-$VERSION
GOOS=linux GOARCH=arm $GOEXEC build -o dist/socksgo-linux-arm-$VERSION
GOOS=linux GOARCH=amd64 $GOEXEC build -o dist/socksgo-linux-amd64-$VERSION
CGO_ENABLED=1 GOOS=android GOARCH=arm GOARM=7 CC=$GOMOBILECC $GOEXEC build -o dist/socksgo-android-arm7-$VERSION
