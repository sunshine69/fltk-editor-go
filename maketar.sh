#!/bin/bash

# Build and create a tarball with name reflecting which current system to build

ARCH=$(uname -m)
OS=$(uname -s)
GO_TAG="icu json1 fts5 secure_delete"
BINARY_NAME="fltkeditor"

if [ "$OS" = "Linux" ]; then
    DISTRO_NAME=$(grep '^NAME=' /etc/os-release | sed 's/ //g;s/"//g' | cut -f2 -d=)
    DISTRO_VER=$(grep '^VERSION_ID=' /etc/os-release | sed 's/ //g;s/"//g' | cut -f2 -d=)
    TARBALL_NAME="${BINARY_NAME}-${DISTRO_NAME}-${DISTRO_VER}-${ARCH}.tgz"
    REDHAT_SUPPORT_PRODUCT_VERSION=$(grep '^REDHAT_SUPPORT_PRODUCT_VERSION=' /etc/os-release | sed 's/ //g;s/"//g' | cut -f2 -d=)
    if [ "$REDHAT_SUPPORT_PRODUCT_VERSION" = "8" ]; then
        GO_TAG="${GO_TAG} pango_1_42 gtk_3_22"
    fi
elif [ "$OS" = "Darwin" ]; then
    ProductName=$( sw_vers | grep ProductName | sed 's/ //g; s/\t//g' | cut -f2 -d: )
    ProductVersion=$( sw_vers | grep ProductVersion | sed 's/ //g; s/\t//g' | cut -f2 -d: )
    TARBALL_NAME="${BINARY_NAME}-${ProductName}-${ProductVersion}-${ARCH}.tgz"
elif [[ "$OS" =~ MINGW64 ]]; then
    go build -ldflags="-s -w -H=windowsgui" --tags "json1 fts5 secure_delete"  -o ${BINARY_NAME}.exe .
    if [ "$1" == "" ]; then
        echo "Enter your mingw64 root dir, example c:/tools/msys64/mingw64: "
        read MINGW64_ROOT_DIR
    else
        MINGW64_ROOT_DIR=$1
    fi
    if [ "$MINGW64_ROOT_DIR" == "" ]; then
        MINGW64_ROOT_DIR="c:/tools/msys64/mingw64"
    fi

    ./${BINARY_NAME}.exe -create-win-bundle "$MINGW64_ROOT_DIR" ${BINARY_NAME}-windows-bundle
    pushd .
    cd ..

    zip -r ${BINARY_NAME}-windows-bundle.zip ${BINARY_NAME}-windows-bundle
    echo "Output bundle file: $(pwd)/${BINARY_NAME}-windows-bundle.zip"
    rm -rf ${BINARY_NAME}-windows-bundle
    popd
    exit 0
fi

go build --tags "${GO_TAG}" -ldflags='-s -w' -o ${BINARY_NAME}

rm -rf ${BINARY_NAME}.app >/dev/null 2>&1
mkdir ${BINARY_NAME}.app
cp -a ${BINARY_NAME} ${BINARY_NAME}.app/
tar czf $TARBALL_NAME ${BINARY_NAME}.app

echo Tar ball pkg is $TARBALL_NAME
