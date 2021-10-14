### Excecution times, 1st block (height 7.5M)

Tickmarks |          ns |         count | total us | Description
----------|-------------|---------------|----------|------------
  1-  0   |     4088134 |             1 |     4088 | **block read**
  2-  1   |         502 |             1 |        0 | is stopped?
  3-  2   |         172 |             1 |        0 | transactions length
  4-  3   |   171200285 |             1 |   171200 | **excecuteBlock**
  5-  4   |         162 |             1 |        0 | add gas
 11- 10   |         119 |          3529 |      420 | stateObject cached?
 13- 12   |      286441 |           292 |    83640 | **read account**
 15- 14   |        1301 |           266 |      346 | add stateObject
 21- 20   |         128 |           188 |       24 | code cached?
 23- 22   |      214896 |            71 |    15257 | **read code**
 31- 30   |          86 |          1106 |       95 | storage dirty?
 33- 32   |          93 |           950 |       88 | storage cached?
 35- 34   |      136394 |           331 |    45146 | **read storage**
 37- 36   |         113 |           331 |       37 | output to int256
 38- 37   |        1092 |           331 |      361 | save to cache

### Excecution times, 10000 blocks (starting from 7.5M), no prefetch

Tickmarks |          ns |         count | total ms | Description
----------|-------------|---------------|----------|------------
  1-  0   |      693265 |         10000 |     6932 | **block read**
  2-  1   |         403 |         10000 |        4 | is stopped?
  3-  2   |         251 |         10000 |        2 | transactions length
  4-  3   |    20895801 |         10000 |   208958 | **excecuteBlock**
  5-  4   |         141 |         10000 |        1 | add gas
 11- 10   |          87 |      38654000 |     3387 | stateObject cached?
 13- 12   |       22004 |       1824923 |    40156 | **read account**
 15- 14   |         677 |       1701345 |     1153 | add stateObject
 21- 20   |          72 |       2258116 |      162 | code cached?
 23- 22   |        4423 |        613418 |     2713 | **read code**
 31- 30   |          78 |      18058060 |     1412 | storage dirty?
 33- 32   |          89 |      14554890 |     1298 | storage cached?
 35- 34   |        9983 |       4345470 |    43383 | **read storage**
 37- 36   |         105 |       4345470 |      459 | output to int256
 38- 37   |         366 |       4345470 |     1593 | save to cache

### Excecution times, 10000 blocks (starting from 7.5M), prefetch blocks (depth=1)

Tickmarks |          ns |         count | total ms | Description
----------|-------------|---------------|----------|------------
  1-  0   |       28213 |         10000 |      282 | **block read**
  2-  1   |         300 |         10000 |        3 | is stopped?
  3-  2   |         215 |         10000 |        2 | transactions length
  4-  3   |    20953124 |         10000 |   209531 | **excecuteBlock**
  5-  4   |         139 |         10000 |        1 | add gas
 11- 10   |          88 |      38654000 |     3408 | stateObject cached?
 13- 12   |       22241 |       1824923 |    40589 | **read account**
 15- 14   |         672 |       1701345 |     1144 | add stateObject
 21- 20   |          72 |       2258116 |      164 | code cached?
 23- 22   |        4500 |        613418 |     2760 | **read code**
 31- 30   |          77 |      18058060 |     1403 | storage dirty?
 33- 32   |          91 |      14554890 |     1324 | storage cached?
 35- 34   |        9997 |       4345470 |    43444 | **read storage**
 37- 36   |         106 |       4345470 |      461 | output to int256
 38- 37   |         361 |       4345470 |     1569 | save to cache


### Excecution times, 10000 blocks (starting from 7.5M), prefetch blocks (from now on depth=101)

Tickmarks |          ns |         count | total ms | Description
----------|-------------|---------------|----------|------------
  1-  0   |        5042 |         10000 |       50 | **block read** (with prefetch)
  2-  1   |         340 |         10000 |        3 | is stopped?
  3-  2   |         211 |         10000 |        2 | transactions length
  4-  3   |    20944515 |         10000 |   209445 | **excecuteBlock**
  5-  4   |         137 |         10000 |        1 | add gas
 11- 10   |          87 |      38654000 |     3378 | stateObject cached?
 13- 12   |       22185 |       1824923 |    40487 | **read account**
 15- 14   |         693 |       1701345 |     1179 | add stateObject
 21- 20   |          71 |       2258116 |      160 | code cached?
 23- 22   |        4413 |        613418 |     2707 | **read code**
 31- 30   |          77 |      18058060 |     1397 | storage dirty?
 33- 32   |          91 |      14554890 |     1326 | storage cached?
 35- 34   |       10017 |       4345470 |    43528 | **read storage**
 37- 36   |         106 |       4345470 |      464 | output to int256
 38- 37   |         362 |       4345470 |     1574 | save to cache

### Excecution times, 10000 blocks (starting from 7.5M), prefetch blocks + from/to accounts

Tickmarks |          ns |         count | total ms | Description
----------|-------------|---------------|----------|------------
  1-  0   |       13044 |         10000 |      130 | **block read** (with prefetch)
  2-  1   |         373 |         10000 |        3 | is stopped?
  3-  2   |         223 |         10000 |        2 | transactions length
  4-  3   |    18354320 |         10000 |   183543 | **excecuteBlock**
  5-  4   |         145 |         10000 |        1 | add gas
 11- 10   |          88 |      38654000 |     3410 | stateObject cached?
 13- 12   |        6587 |       1824923 |    12021 | **read account** (with prefetch)
 15- 14   |         639 |       1701345 |     1088 | add stateObject
 21- 20   |          75 |       2258116 |      171 | code cached?
 23- 22   |        4576 |        613418 |     2807 | **read code**
 31- 30   |          78 |      18058060 |     1411 | storage dirty?
 33- 32   |          91 |      14554890 |     1336 | storage cached?
 35- 34   |       10378 |       4345470 |    45101 | **read storage**
 37- 36   |         107 |       4345470 |      465 | output to int256
 38- 37   |         366 |       4345470 |     1593 | save to cache

### Excecution times, 10000 blocks (starting from 7.5M), prefetch blocks + from/to accounts + code

Tickmarks |          ns |         count | total ms | Description
----------|-------------|---------------|----------|------------
  1-  0   |       12437 |         10000 |      124 | **block read** (with prefetch)
  2-  1   |         284 |         10000 |        2 | is stopped?
  3-  2   |         244 |         10000 |        2 | transactions length
  4-  3   |    18210844 |         10000 |   182108 | **excecuteBlock**
  5-  4   |         138 |         10000 |        1 | add gas
 11- 10   |          89 |      38654000 |     3445 | stateObject cached?
 13- 12   |        6477 |       1824923 |    11821 | **read account** (with prefetch)
 15- 14   |         650 |       1701345 |     1106 | add stateObject
 21- 20   |          72 |       2258116 |      164 | code cached?
 23- 22   |        3080 |        613418 |     1889 | **read code** (with prefetch)
 31- 30   |          78 |      18058060 |     1414 | storage dirty?
 33- 32   |          90 |      14554890 |     1318 | storage cached?
 35- 34   |       10364 |       4345470 |    45038 | **read storage**
 37- 36   |         106 |       4345470 |      463 | output to int256
 38- 37   |         369 |       4345470 |     1604 | save to cache

### Speedup

Test               | Seconds | Speedup
-------------------|---------|--------
no prefetch        |  215.9  |  1.00x
1 block only       |  209.8  |  1.03x
block (depth=101)  |  209.5  |  1.03x
block+account      |  183.7  |  1.18x
block+account+code |  182.2  |  1.18x

### Excecution times, 10000 blocks (starting from 7.5M), hot FS cache

Tickmarks |          ns |         count | total ms | Description
----------|-------------|---------------|----------|------------
  1-  0   |        3909 |         10000 |       39 | **block read** (with prefetch)
  2-  1   |         233 |         10000 |        2 | is stopped?
  3-  2   |         219 |         10000 |        2 | transactions length
  4-  3   |    14280324 |         10000 |   142803 | **excecuteBlock**
  5-  4   |         142 |         10000 |        1 | add gas
 11- 10   |          87 |      38654000 |     3397 | stateObject cached?
 13- 12   |        4277 |       1824923 |     7806 | **read account** (with prefetch)
 15- 14   |         713 |       1701345 |     1213 | add stateObject
 21- 20   |          69 |       2258116 |      156 | code cached?
 23- 22   |        2936 |        613418 |     1801 | **read code** (with prefetch)
 31- 30   |          77 |      18058060 |     1406 | storage dirty?
 33- 32   |          90 |      14554890 |     1322 | storage cached?
 35- 34   |        2695 |       4345470 |    11714 | **read storage**
 37- 36   |         107 |       4345470 |      466 | output to int256
 38- 37   |         381 |       4345470 |     1659 | save to cache

maximum potential speedup: 1.51x (142.8 seconds)
