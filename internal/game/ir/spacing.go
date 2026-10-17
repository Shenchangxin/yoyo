package ir

import "math"

func ValidateSpacing(els []Placement) error {
	for i := 0; i < len(els); i++ {
		for j := i + 1; j < len(els); j++ {
			if LogicalOnly(els[i].ElementType) || LogicalOnly(els[j].ElementType) {
				continue
			}
			if !RequiresVisibleAsset(els[i].ElementType) || !RequiresVisibleAsset(els[j].ElementType) {
				continue
			}
			d := math.Hypot(els[i].WorldXYZ[0]-els[j].WorldXYZ[0], els[i].WorldXYZ[2]-els[j].WorldXYZ[2])
			if d < DefaultMinFootprintGap {
				return errf("placements %s and %s are closer than %.2fm", els[i].PlacementID, els[j].PlacementID, DefaultMinFootprintGap)
			}
		}
	}
	return nil
}
