
import json


def read_input_file():
    print('Reading')
    data  = {}
    total = {}
    calls = {}
    with open('jump_counts.txt') as f:
        i    = -1
        line = ''
        try:
            m1 = {}
            m2 = {}
            m3 = {}
            for i, line in enumerate(f):
                line = line[:-1]
                #
                if   line[0] != ' ':
                    ps       = line.split()
                    assert len(ps) == 3
                    _h       =     ps[0]
                    c        = int(ps[1])
                    t        = int(ps[2])
                    assert len(_h) == 66
                    assert _h[:2] == 'h_'
                    h        = bytes.fromhex(_h[2:])
                    if len(data) >= 100:
                        print('Read', len(data), 'contracts')
                        yield data, total, calls
                        data.clear()
                        total.clear()
                        calls.clear()
                        print('Reading')
                    m1       = {}
                    data[h]  = m1
                    total[h] = t
                    calls[h] = c
                elif line[1] != ' ':
                    assert len(line) == 7
                    src     = int(line, 16)
                    m2      = {}
                    m1[src] = m2
                elif line[2] != ' ':
                    assert len(line) == 8
                    dst     = int(line, 16)
                    m3      = {}
                    m2[dst] = m3
                else:
                    ps      = line.split()
                    assert len(ps) == 2
                    cid     = int(ps[0])
                    c       = int(ps[1])
                    m3[cid] = c
                #
        except Exception as e:
            print('At', i, line, repr(e))
            raise


res = {}


for data, total, calls in read_input_file():

    print('Verifying sums')

    for h, m1 in data.items():
        assert sum(
            sum(
                sum(m3.values())
                for m3 in m2.values()
            )
            for m2 in m1.values()
        ) == total[h]
        assert len(set.union(*(
            set.union(*(
                set(m3.keys())
                for m3 in m2.values()
            ))
            for m2 in m1.values()
        ))) == calls[h]

    print('Verified')

    print('Transforming')

    MIN_CALLS = 50

    # callsn = {}

    # for h, m1 in data.items():
    #     callsn[h] = len(set.union(*(
    #         set.union(set(), *(
    #             set(m3.keys())
    #             for m3 in m2.values()
    #             if len(m3) >= MIN_CALLS
    #         ))
    #         for m2 in m1.values()
    #     )))

    # import numpy
    # import matplotlib.pyplot as plt

    # ef  = {}
    # ecf = {}

    # def cumsum_norm_rev(ar):
    #     ar    = ar.cumsum()
    #     scale = ar[-1]
    #     return (scale - ar) / scale

    # for h, m1 in data.items():
    #     ar_f   = numpy.zeros(total[ h] + 1)
    #     ar_cf  = numpy.zeros(calls[h] + 1)
    #     for m2 in m1.values():
    #         for m3 in m2.values():
    #             ar_f[sum(m3.values())] += 1
    #             ar_cf[len(m3)]         += 1
    #     ef[h]  = cumsum_norm_rev(ar_f)
    #     ecf[h] = cumsum_norm_rev(ar_cf)

    # for h, m1 in data.items():
    #     y = ef[h]
    #     x = numpy.arange(len(y)) + 1
    #     plt.plot(x, y)

    # plt.show()

    # for h, m1 in data.items():
    #     y = ecf[h]
    #     x = numpy.arange(len(y)) + 1
    #     plt.plot(x, y)

    # plt.show()

    # stats = {}

    for h, m1 in data.items():
        # sm1 = {}
        # ec  = 0
        # ecn = 0
        for src, m2 in list(m1.items()):
            m2n = {
                dst: (len(m3), sum(m3.values()))
                for dst, m3 in m2.items()
                if len(m3) >= MIN_CALLS
            }
            m1[src]  = m2n
            # sm1[src] = (
            #     len(set.union(*(
            #         set(m3.keys())
            #         for m3 in m2.values()
            #     ))),
            #     len(set.union(set(), *(
            #         set(m3.keys())
            #         for m3 in m2.values()
            #         if len(m3) >= MIN_CALLS
            #     ))),
            #     sum(
            #         sum(m3.values())
            #         for m3 in m2.values()
            #     ),
            #     sum(
            #         s
            #         for _, s in m2n.values()
            #     ),
            #     len(m2),
            #     len(m2n),
            # )
            # ec  += len(m2)
            # ecn += len(m2n)
        # stats[h] = (sm1, ec, ecn)

    print('Transformed')

    print()

    if False:
        for h, m1 in data.items():
            sm1, ec, ecn = stats[h]
            print()
            print()
            print('h_' + h.hex(), total[h], calls[h], callsn[h], ec, ecn)
            print(' SRC   ->  DST   CALLS   SUM')
            for src, m2 in sorted(list(m1.items())):
                c, cn, s, sn, l, ln = sm1[src]
                if   len(m2) == 0: continue
                elif len(m2) == 1 and False:
                    dst, (c, s) = list(m2.items())[0]
                    print(f'{src:6x} -> {dst:6x} {c:5d} {s:5d} {c:4d} {cn:4d} {s:4d} {sn:4d} {l:2d} {ln:2d}')
                else:
                    print(f'{src:6x}                       {    c:4d} {cn:4d} {s:4d} {sn:4d} {l:2d} {ln:2d}')
                    for dst, m3 in sorted(list(m2.items())):
                        c, s = m3
                        print(f"    '---> {dst:6x} {c:5d} {s:5d}")

    if False:
        for h, m1 in data.items():
            print()
            print()
            print('h_' + h.hex())
            print(' SRC   ->  DST   CALLS   SUM')
            for src, m2 in sorted(list(m1.items())):
                if   len(m2) == 0: continue
                elif len(m2) == 1:
                    dst, (c, s) = list(m2.items())[0]
                    print(f'{src:6x} -> {dst:6x} {c:5d} {s:5d}')
                else:
                    print(f'{src:6x}')
                    for dst, m3 in sorted(list(m2.items())):
                        c, s = m3
                        print(f"    '---> {dst:6x} {c:5d} {s:5d}")

    if False:
        for h, m1 in data.items():
            print()
            print('h_' + h.hex())
            for src, m2 in sorted(list(m1.items())):
                for dst in sorted(list(m2)):
                    print(f'{src:6x} {dst:6x}')

    if True:
        for h, m1 in data.items():
            res[h] = json.dumps([
                (src, dst)
                for src, m2 in m1.items()
                for dst in m2
            ], separators=(',', ':'))
            # json.dump({
            #     src: [
            #         dst
            #         for dst in m2
            #     ]
            #     for src, m2 in m1.items()
            #     if m2
            # }, output_file)


with open('jump_edges.json', 'w') as fo:
    fo.write(
        '{\n' + 
        ',\n'.join(
            fo.write('"h_' + h.hex() + '":' + line)
            for h, line in sorted(list(res.items()))
        ) +
        '\n}\n'
    )
