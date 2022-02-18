
import sys

Mi = 1024 * 1024
BS = 2 * Mi

extras  = b'Tx \n'
ascii_0 = b'0'[0]
ascii_9 = b'9'[0]
ascii_a = b'a'[0]
ascii_f = b'f'[0]

def is_bad(b):
    if ascii_0 <= b <= ascii_9: return False
    if ascii_a <= b <= ascii_f: return False
    if b in extras:             return False
    return True

def main(fname):
    with open(fname, 'rb') as f:
        offs = 0
        while True:
            data = f.read(BS)
            if not data: break
            for b in filter(is_bad, data):
                print('Problem after', offs, 'seen', b, repr(bytes([b])))
                return
            # for b in data:
            #     if is_bad(b):
            #         print('Problem after', offs, 'seen', b, repr(bytes([b])))
            #         return
            offs += BS
            print(f'\rNow at {offs//1024//1024:5} MiB', end='')
    print()


if __name__ == '__main__':
    fname = sys.argv[1]
    main(fname=fname)
