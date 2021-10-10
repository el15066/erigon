
prefetches = []
with open('prefetches.txt') as f:
    for line in f:
        ps = line.split()
        prefetches.append(ps[0] + ps[-1])

reads = []
with open('reads.txt') as f:
    for line in f:
        ps = line.split()
        reads.append(ps[0] + ps[-1])

print( 'Metric            | Result')
print( '------------------|--------')
print( 'reads             |', len(reads))
print( 'prefetches        |', len(prefetches))

sreads      = set(reads)
sprefetches = set(prefetches)

print(f'unique reads      | {100 * len(sreads)      / len(reads)     :5.1f}%')
print(f'unique prefetches | {100 * len(sprefetches) / len(prefetches):5.1f}%')

print(f'reads prefetched  | {100 - 100 * len(sreads - sprefetches) / len(sreads)    :5.1f}% of total (unique) reads')
print(f'false positives   | {      100 * len(sprefetches - sreads) / len(prefetches):5.1f}% of total (unique) prefetches')
