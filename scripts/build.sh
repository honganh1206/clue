#!/bin/bash

# ./scripts/build.sh <version-number>
build() {
    os=$1
    app_name=$2
    version=$3
    arch=$4
    # Shortened version of git commit hash
    sha1=$(git rev-parse --short HEAD | tr -d '\n')

    output_name="${app_name}_${version}_${os}_${arch}"
    if [[ "$os" == "windows" ]]; then
      output_name="${output_name}.exe"
    fi

    echo "Building for $os/$arch (version: $version) -> $output_name"
    # TODO: On darwin/macos we need clang
    # On windows gcc does not recognize -mthreads and it must be -pthread
    CGO_ENABLED=1 GOOS=$os GOARCH=$arch go build -o "dist/${version}/${output_name}" -ldflags "-X main.sha1ver=$sha1" main.go
}

targets=(
    # "darwin"
    "linux"
    # "windows"
)
app_name="clue"

if [ $# -eq 0 ]; then
  echo "Error: Please provide a version number as a command-line argument."
  exit 1
fi

version=$1

# Clear old build artifacts
rm -rf dist

# Create the output directory
mkdir dist

# Build for each OS
for os in "${targets[@]}"; do
  # build $os $app_name $version "arm64"
  build $os $app_name $version "amd64"
done

echo "Done!"
