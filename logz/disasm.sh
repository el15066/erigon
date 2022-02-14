#!/bin/bash -e

disasm() {
    fname="$(basename "$1" | cut -d '.' -f 1)"
    printf "\r$fname "
    evmasm -d -i "code/$fname.runbin.hex" -o "disasm/$fname.disasm"
}

if [ "$1" == "inner" ]; then
    disasm "$2"
elif [ "$1" ]; then
    echo "Usage:" "$0"
    exit 1
else
    mkdir disasm                           || true
    printf 'Deleting *.disasm files in disasm/ : '
    find  disasm/ -name '*.disasm' | wc -l || true
    find  disasm/ -name '*.disasm' -delete || true
    #find  code/   -name '*.runbin.hex' -exec "$0" inner {} \;
    find  code/   -name '*.runbin.hex' | sort | while read -r fname
    do
        disasm "$fname"
    done
    echo $'\n''All files in code/ have been disassembled. The output files are in disasm/ .'
fi
