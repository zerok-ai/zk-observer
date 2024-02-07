#!/bin/sh

while [[ $# -gt 0 ]]; do
    case "$1" in
        --amd64)
            exeAMD64="$2"
            shift 2
            ;;
        --arm64)
            exeARM64="$2"
            shift 2
            ;;
        -c)
            config="$2"
            shift 2
            ;;
        *)
            echo "Usage: $0"
            echo "    --amd64   path of amd64 executable"
            echo "    --arm64   path of arm64 executable"
            echo "    -c        path to config file"
            exit 1
            ;;
    esac
done

arch=$(uname -m)

if [ "$arch" = "x86_64" ] || [ "$arch" = "amd64" ]; then
    echo "This is an AMD64 machine."
    echo $PWD
    ./"$exeAMD64" -c "$config"
elif [ "$arch" = "aarch64" ] || [ "$arch" = "arm64" ]; then
    echo "This is an ARM64 machine."
    echo $PWD
    ./"$exeARM64" -c "$config"
else
    echo "Unsupported Architecture: $arch"
fi