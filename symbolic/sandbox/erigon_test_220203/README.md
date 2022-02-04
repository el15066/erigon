
#### 20220203
Athlon 200GE
B450M
62 GiB DDR4
sx8200 2TB nvme ext4

##### Times

###### No prefetch
Tickmarks | total ms | Description
----------|----------|----------------------
   1-   0 |     6421 | **block read/fetch**
   4-   3 |   268689 | **execute block**

###### Prefetch blocks
Tickmarks | total ms | Description
----------|----------|----------------------
   1-   0 |       81 | **block read/fetch**
   4-   3 |   268338 | **execute block**


###### Prefetch blocks + from/to accounts
Tickmarks | total ms | Description
----------|----------|----------------------
   1-   0 |      140 | **block read/fetch**
   4-   3 |   233976 | **execute block**

###### Prefetch blocks + from/to accounts + code
Tickmarks | total ms | Description
----------|----------|----------------------
   1-   0 |      161 | **block read/fetch**
   4-   3 |   232813 | **execute block**

###### Prefetch blocks + from/to accounts + code + run predictors
Tickmarks | total ms | Description
----------|----------|----------------------
   1-   0 |      357 | **block read/fetch**
   4-   3 |   217524 | **execute block**

###### Hot FS cache
Tickmarks | total ms | Description
----------|----------|----------------------
   1-   0 |     2269 | **block read/fetch**
   4-   3 |   184052 | **execute block**

###### Hot FS cache + Prefetch blocks
Tickmarks | total ms | Description
----------|----------|----------------------
   1-   0 |       78 | **block read/fetch**
   4-   3 |   184733 | **execute block**

###### Hot FS cache + Prefetch * + run predictors
Tickmarks | total ms | Description
----------|----------|----------------------
   1-   0 |      140 | **block read/fetch**
   4-   3 |   188265 | **execute block**

##### Speedup

Test            | Seconds | Speedup
----------------|---------|--------
no prefetch     |    275  | 1.00
block           |    268  | 1.03
block+accs      |    234  | 1.18
block+accs+code |    233  | 1.18
* + pred        |    218  | 1.26
hot             |    186  | 1.48
hot+blocks      |    185  | 1.49
hot + * + pred  |    188  | 1.46
