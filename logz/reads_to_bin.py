
def processline(line):
  ps  = line.split()
  bid = int(ps[1])
  tid = int(ps[2])
  a   = bytes.fromhex(ps[3])
  ca  = a[  :28] # contract address + incarnation
  sa  = a[28:60] # storage  address
  return bid, tid, ca, sa

bid = 7500000
tid = 0
with open('reads.txt') as fi:
  with open('reads_s.bin', 'wb') as fo:
    # fo.write(bytes.fromhex('0000'))
    line = fi.readline()
    while line:
      if line[0] != 'S':
        line = fi.readline()
        continue
      nbid, ntid, nca, sa = processline(line)
      if nbid != bid:
        fo.write(bytes.fromhex('00000000')+(nbid-bid).to_bytes(2,'big'))
        bid = nbid
        tid = 0
      if ntid != tid:
        fo.write(bytes.fromhex('0000')+(ntid-tid).to_bytes(2,'big'))
        tid = ntid
      ca    = nca
      addrs = []
      while line:
        nbid, ntid, nca, sa = processline(line)
        if (nbid, ntid, nca) != (bid, tid, ca): break
        addrs.append(sa)
        line = fi.readline()
      fo.write(len(addrs).to_bytes(2,'big')+ca+b''.join(addrs))
