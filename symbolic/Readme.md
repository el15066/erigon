## Symbolic execution / static analysis

Μέχρι τώρα έχω δοκιμάσει το [manticore](https://github.com/trailofbits/manticore) και το [mythril](https://github.com/ConsenSys/mythril/tree/v0.22.27).

Το πρώτο έβγαζε exception στις δοκιμές που έκανα και το άφησα για τώρα. Επίσης, στο orbs token contract που το άφησα παρά τα exception πήρε συνολικά καμιά ώρα για την default ανάλυση που κάνει. Δεν βρήκα option να του δόσω το bytecode κατευθείαν οπότε του έδωσα το source και έκανε αυτό compile.

Το δεύτερο [ανέλυσε γρήγορα το orbs](sandbox/myth_orbs_0/), έχει και option για κατευθείαν bytecode, οπότε έκανα παραπάνω δοκιμές. Πήρα τα 10 contracts που έχουν κληθεί απευθείας πιο συχνά στα block [7_000_000,7_500_000) και του έδωσα να κάνει τη default ανάλυση 'safe-functions' (μόνο για να δω πόσο χρόνο θα πάρει). Στα 6/10 που είχε πάρει πάνω από 5\~6 λεπτά το σταμάτησα, στα άλλα 4 τελείωσε εντός 4 λεπτών. Αυτά τα 4 βέβαια είναι παρόμοια (ERC20), αλλά είχαν σημαντική διακύμανση στο χρόνο που πήραν. Αποτελέσματα [εδώ](sandbox/top10_runbins.log) και [στα .log εδώ](sandbox/top10/).

Θα κοιτάξω πιο αναλυτικά το mythril για να δω τι ακριβώς κάνει και πως θα μπορούσε να τροποποιηθεί, μιας και δεν είνα η ανάλυση που κάνει αυτό που θέλουμε.

#### Update 30/10/21

Έχω δοκιμάσει και το [rattle (fork)](https://github.com/el15066/rattle). (στο fork έχω κάνει και κάποιες τροποποιήσεις)
Είναι πιο απλό από τα προηγούμενα, κάνοντας κυρίως μετατροπή του bytecode σε μορφή [SSA](https://en.wikipedia.org/wiki/Static_single_assignment_form) και (optional) κάποια optimization (πχ constant folding). Στην έξοδό του βγάζει και το control flow graph σε εικόνες. Άρα κυρίως κάνει στατική ανάλυση σαν τους compilers.
Με τον [original κώδικα](https://github.com/crytic/rattle) έχω κάνει 2 δοκιμές στο orbs token (ERC20), [μία χωρίς](sandbox/rat_orbs_0/) και [μία με](sandbox/rat_orbs_0/). Επίσης, στο [sandbox/rat_top10_runbins.log](sandbox/rat_top10_runbins.log) και [sandbox/rat_top10/](sandbox/rat_top10/) είναι τα αποτελέσματα (με optimization) για τα ίδια top10 contracts που ανέλυσα με το mythril πριν ([sandbox/top10_runbins.log](sandbox/top10_runbins.log), [sandbox/top10/](sandbox/top10/)). 5 απ'τα 10 απέτυχαν λόγω αναδρομής (recursion depth exceeded).
Στο fork έχω προσθέσει τη δυνατότητα να αναλύει και κάποιες περιπτώσεις πρόσβασης στη μνήμη (διαφορετικά αναλύει μόνο τη στοίβα), καθώς και την εκτύπωση εντολών ανά τύπο μαζί με το δέντρο από data dependencies που έχουν.

##### Παράδειγμα εξόδου για την συνάρτηση transfer() του ERC20 token:

+ Original χωρίς optimization
![](sandbox/rat_orbs_0/output/transfer(address,uint256).png)

+ Original με optimization
![](sandbox/rat_orbs_1/output/transfer(address,uint256).png)

+ Με optimization + απλό memory tracking
![](sandbox/rat_mem_test_orbs_1/output/transfer(address,uint256).png)
*(Το `SHA3` πιο πριν έπερνε διεύθυνση μνήμης σαν arguement, το `SHA3RES` εδώ παίρνει κατευθείαν τα data, τα οποία τυπικά πρέπει πρώτα να γίνουν concatenate. Σε comment είναι η διεύθυνση μνήμης που έπερνε αρχικά).*

+ Στο τελευταίο, [στο τέλος του log](sandbox/rat_mem_test_orbs_1/log.txt#L1425) μπορούμε να δούμε και τις εντολές `CALL` και `SLOAD` που βρήκε:  
`python3 ~/el15066/rattle/rattle-cli.py --optimize --input ../contracts/orbs.runbin.hex --pick 'SLOAD,CALL' |& tee log.txt`
```
[...]
---- Picking SLOAD    instrunctions ----
SLOAD
 '- SHA3RES
     |- AND
     |   |- #ffffffffffffffffffffffffffffffffffffffff
     |   '- AND
     |       |- #ffffffffffffffffffffffffffffffffffffffff
     |       '- <Unresolved sp:-4 block:0x14c3> // <---- αυτό είναι αποτυχία εύρεσης της παραμέτρου
     '- SHA3RES
         |- CALLER
         '- #3
SLOAD
 '- #0
SLOAD
 '- SHA3RES
     |- CALLDATALOAD
     |   '- #24
     '- SHA3RES
         |- CALLDATALOAD
         |   '- #4
         '- #3
SLOAD
 '- SHA3RES
     |- CALLDATALOAD
     |   '- #4
     '- SHA3RES
         |- CALLER
         '- #3
SLOAD
 '- #2
SLOAD
 '- SHA3RES
     |- CALLDATALOAD
     |   '- #4
     '- #1
SLOAD
 '- SHA3RES
     |- CALLER
     '- #1
SLOAD
 '- SHA3RES
     |- CALLER
     '- SHA3RES
         |- CALLDATALOAD
         |   '- #4
         '- #3
----------------------------------------
---- Picking CALL     instrunctions ----
CALL
 |- GAS
 |- AND
 |   |- #ffffffffffffffffffffffffffffffffffffffff
 |   '- <Unresolved sp:-3 block:0x180a>
 |- #0
 |- MLOAD
 |   '- #40
 |- SUB
 |   |- ADD
 |   |   |- #20
 |   |   '- ADD
 |   |       |- #20
 |   |       '- ADD
 |   |           |- #4
 |   |           '- MLOAD
 |   |               '- #40
 |   '- MLOAD
 |       '- #40
 |- MLOAD
 |   '- #40
 '- #20
CALL
 |- GAS
 |- CALLDATALOAD
 |   '- #4
 |- #0
 |- MLOAD
 |   '- #40
 |- SUB
 |   |- ADD
 |   |   |- #20
 |   |   '- ADD
 |   |       |- #4
 |   |       '- MLOAD
 |   |           '- #40
 |   '- MLOAD
 |       '- #40
 |- MLOAD
 |   '- #40
 '- #0
CALL
 |- GAS
 |- CALLDATALOAD
 |   '- #4
 |- #0
 |- MLOAD
 |   '- #40
 |- SUB
 |   |- ADD
 |   |   |- #20
 |   |   '- ADD
 |   |       |- #4
 |   |       '- MLOAD
 |   |           '- #40
 |   '- MLOAD
 |       '- #40
 |- MLOAD
 |   '- #40
 '- #20
----------------------------------------
```
