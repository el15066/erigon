# # Convert from:
# S  7500000   6 ff56cc6b1e6ded347aa0b7676c85ab0b3d08b0fa000000000000000178802c4c4a0b1949a61c08b716166e1c2adf16155653039ded4304c118fbc908
# S  7500000   6 ff56cc6b1e6ded347aa0b7676c85ab0b3d08b0fa000000000000000145cfa2ae977bd9599e849db8b8a7e047bbdcbd16355bb69707961646d7a189c9 0123
# # To:
# Tx  7500000   6


def processline(line):
    assert len(line) >= 136
    assert line[:2] == 'S '
    bid  = line[  2: 10]
    assert line[10] == ' '
    tid  = line[ 11: 14]
    assert line[14] == ' '
    a_lf = line[ 71:135] + '\n'
    #assert line[135] == '\n'
    return bid, tid, a_lf

bid, tid, alfs = '', '', set()
with open('reads.txt') as fi:
    with open('reads_addresses_by_tx.txt', 'w') as fo:
        for i, line in enumerate(fi):
            # if i & 0x07ffff == 0: print(f'Input at {i*136>>20:5} MiB', end='\r') # no longer exact
            if i & 0x07ffff == 0: print(f'Input at line {i>>10:7} Ki', end='\r')
            if line[0] == 'S':
                nbid, ntid, a_lf = processline(line)
                if (nbid, ntid) != (bid, tid):
                    for aa in sorted(list(alfs)): fo.write(aa)
                    bid, tid, alfs = nbid, ntid, set()
                    fo.write(f'Tx {bid} {tid}\n')
                alfs.add(a_lf)
                # if a_lf not in alfs:
                #     alfs.add(a_lf)
                #     fo.write(a_lf)
        for aa in sorted(list(alfs)): fo.write(aa)

print('Complete! output is at reads_addresses_by_tx.txt')
