
### Execution errors

Result              | # of TXs
--------------------|---------
Unknown instruction | 13385
Memory > 4 KiB      | 103
Unknown jump target | 11531
No error            | 63807
Total               | 88826

#### Unknown instructions

Name        | #    | Fixed
------------|------|------
BALANCE     | 3276 | todo (fast)
CODECOPY    | 814  | todo (fast)
EXTCODESIZE | 6403 | todo (slow)
GASPRICE    | 1932 | todo (trivial)
NUMBER      | 960  | done
