#!/bin/bash

# ./scripts/build.sh (reads version from VERSION file)
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
    CGO_ENABLED=1 GOOS=$os GOARCH=$arch go build -o "dist/${version}/${output_name}" -ldflags "-X github.com/honganh1206/clue/cmd.Version=$version -X github.com/honganh1206/clue/cmd.GitCommit=$sha1" main.go
}

targets=(
    # "darwin"
    "linux"
    # "windows"
)
app_name="clue"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
version=$(cat "$SCRIPT_DIR/../VERSION" 2>/dev/null)

if [ -z "$version" ]; then
  echo "Error: Could not read version from VERSION file."
  exit 1
fi

echo "Building version: $version"

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
