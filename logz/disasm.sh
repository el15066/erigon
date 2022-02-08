#!/bin/bash -e

disasm() {
    fname="$(basename "$1" | cut -d '.' -f 1)"
    evmasm -d -i "code/$fname.runbin.hex" -o "disasm/$fname.disasm"
}

if [ "$1" == "inner" ]; then
    disasm "$2"
elif [ "$1" ]; then
    echo "Usage:" "$0"
    exit 1
else
    rm -r disasm/ || true
    mkdir disasm  || true
    find code/ -name '*.runbin.hex' -exec "$0" inner {} \;
fi
