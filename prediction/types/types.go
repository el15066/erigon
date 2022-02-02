
package prediction_types

type BlockTableEntry struct {
    Index int
    Edges []uint16 // allow in-edges
}
type BlockTable map[uint16]BlockTableEntry

type Predictor struct {
    BlockTbl BlockTable
    Code     []byte
}
